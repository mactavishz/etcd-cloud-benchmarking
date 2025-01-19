#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then
  set -o xtrace
fi

# Usage functions
print_usage() {
  echo "Usage: $0 [options]"
  echo "Options:"
  echo "  -n, --num_nodes <number>  The number of nodes in the etcd cluster. Default: 1"
  echo "  -z  --zone <zone>         The GCP zone in which the instances are deployed, available options: a, b, c. Default: c"
  echo "  -s, --scenario <name>     The benchmark scenario name to run, available options: kv, lock. Default: kv"
  echo "  -w  --workload <name>     The workload type in kv/lock scenario to run, available options for scenario kv: read-only, read-heavy, update-heavy, available options for scenario lock: lock-only, lock-mixed-read, lock-mixed-write, lock-contention. Multiple values are provided with commas in between, or all can be used for sepcifying all workloads. Default: all"
  echo "  -d  --out_dir <path>      The output directory for the benchmark results, can be relative or absolute path. Default: results"
  exit 0
}

# default options
REGION="europe-central2"
ZONE="${REGION}-c"
SCENARIO="kv"
NUM_NODES=1
OUT_DIR="results"
declare -a WORKLOAD_TYPES=()
declare -a KV_WORKLOAD_TYPES=(
  "read-only"
  "read-heavy"
  "update-heavy"
)
declare -a LOCK_WORKLOAD_TYPES=(
  "lock-only"
  "lock-mixed-read"
  "lock-mixed-write"
  "lock-contention"
)

# machine related variables
ETCD_DATA_DIR="/var/lib/etcd/data"
ETCD_CLIENT_PORT="2379"
BENCHMARK_CLIENT_INSTANCE="benchmark-client"
SYSTEM_SERVICE_NAME="etcd.service"
BENCHMARK_CONTROL_BIN="benchctl"
BENCHMARK_CONFIG_TEMPLATE_DIR="./templates"
BENCHMARK_DATA_DIR="benchmark-data"
BENCHMARK_CLIENT_GRPC_PORT="50051"

# verify etcd node health
verify_node_health() {
  local instance=$1
  gcloud compute ssh "${instance}" --zone=${ZONE} --command="
        ETCDCTL_API=3 etcdctl endpoint health -w table
    "
}

# verify etcd cluster membership
verify_cluster_membership() {
  local instance=$1
  gcloud compute ssh "${instance}" --zone=${ZONE} --command="
        ETCDCTL_API=3 etcdctl member list -w table
    "
}

get_instance_private_ip() {
  local instance_name=$1
  gcloud compute instances describe "$instance_name" --zone="$ZONE" --format="value(networkInterfaces[0].networkIP)"
}

get_instance_public_ip() {
  local instance_name=$1
  gcloud compute instances describe "$instance_name" --zone="$ZONE" --format="value(networkInterfaces[0].accessConfigs[0].natIP)"
}

start_benchmark_client_service() {
  gcloud compute ssh "${BENCHMARK_CLIENT_INSTANCE}" --zone="$ZONE" --command="
  cd $BENCHMARK_DATA_DIR
  rm -rf *
  echo \$(pwd)
  tmux new-session -d -s benchclient 'benchclient -p $BENCHMARK_CLIENT_GRPC_PORT; exit'
  "
}

download_benchmark_client_output_files() {
  local local_dir=$1
  gcloud compute scp --zone="$ZONE" --recurse "${BENCHMARK_CLIENT_INSTANCE}":"$BENCHMARK_DATA_DIR/*" "$local_dir"
}

cleanup_benchmark_client_output_files() {
  gcloud compute ssh "${BENCHMARK_CLIENT_INSTANCE}" --zone="$ZONE" --command="
  rm -rf $BENCHMARK_DATA_DIR/*
  "
}

reset_all_etcd_nodes() {
  local prefix="etcd"
  local node_count=$NUM_NODES
  local command="
  sudo systemctl stop $SYSTEM_SERVICE_NAME
  sudo rm -rf $ETCD_DATA_DIR/*
  "
  case $node_count in
  1) prefix="etcd-single" ;;
  3) prefix="etcd-3" ;;
  5) prefix="etcd-5" ;;
  esac
  if [ "$node_count" -eq 1 ]; then
    gcloud compute ssh "${prefix}" --zone="$ZONE" --command="$command"
    return
  fi
  for i in $(seq 0 $((node_count - 1))); do
    local instance_name="${prefix}-${i}"
    gcloud compute ssh "$instance_name" --zone="$ZONE" --command="$command"
  done
}

restart_all_etcd_nodes() {
  local prefix="etcd"
  local node_count=$NUM_NODES
  local command="
  sudo systemctl start $SYSTEM_SERVICE_NAME
  "
  case $node_count in
  1) prefix="etcd-single" ;;
  3) prefix="etcd-3" ;;
  5) prefix="etcd-5" ;;
  esac
  if [ "$node_count" -eq 1 ]; then
    gcloud compute ssh "${prefix}" --zone="$ZONE" --command="$command"
  else
    for i in $(seq 0 $((node_count - 1))); do
      local instance_name="${prefix}-${i}"
      (
        gcloud compute ssh "$instance_name" --zone="$ZONE" --command="$command" &
      )
    done
  fi

  # Wait for all nodes to restart
  wait

  # Give the cluster some time to establish connections
  echo "Waiting for cluster to stabilize..."
  sleep 20

  # Verify cluster health for all nodes
  if [ "$node_count" -eq 1 ]; then
    verify_node_health "${prefix}"
  else
    for i in $(seq 0 $((node_count - 1))); do
      local instance_name="${prefix}-${i}"
      echo "Verifying health of node: ${instance_name}"
      verify_node_health "${instance_name}"
      if [ "$i" -eq "$((node_count - 1))" ]; then
        # only verify cluster membership for the last node
        verify_cluster_membership "${instance}"
      fi
    done
  fi
}

run_benchmark() {
  echo "Running benchmark with the following options:"
  echo "  - Number of nodes: $NUM_NODES"
  echo "  - GCP zone: $ZONE"
  echo "  - Scenario: $SCENARIO"
  echo "  - Workloads: ${WORKLOAD_TYPES[*]}"
  echo "  - Output directory: $OUT_DIR"

  # check if the binary exists
  if [ ! -x "$(command -v $BENCHMARK_CONTROL_BIN)" ]; then
    echo "Error: $BENCHMARK_CONTROL_BIN binary not found, please build the binary first, check the README for more information"
    exit 1
  fi

  local node_count=$NUM_NODES
  local prefix="etcd"
  local config_file="$BENCHMARK_CONFIG_TEMPLATE_DIR/$SCENARIO.json"
  local instances=()
  local endpoints=""
  local benchmark_client_pubic_ip=$(get_instance_public_ip "${BENCHMARK_CLIENT_INSTANCE}")

  # Set the appropriate instance prefix based on node count
  case $node_count in
  1) prefix="etcd-single" ;;
  3) prefix="etcd-3" ;;
  5) prefix="etcd-5" ;;
  esac

  # loop through the workloads to benchmark
  for workload in "${WORKLOAD_TYPES[@]}"; do
    echo "Running benchmark for workload type: $workload"
    local output_dir="$OUT_DIR/$workload"
    mkdir -p "$output_dir"
    # init the benchmark config
    $BENCHMARK_CONTROL_BIN config init
    # load the benchmark template config file
    $BENCHMARK_CONTROL_BIN config load-file "$config_file"
    # set the workload type
    $BENCHMARK_CONTROL_BIN config set workload_type="$workload"

    # get the instances and endpoints
    for i in $(seq 0 $((node_count - 1))); do
      local instance
      if [ $node_count -eq 1 ]; then
        instance="${prefix}"
      else
        instance="${prefix}-${i}"
      fi
      instances+=("$instance")

      local ip=$(get_instance_private_ip "${instance}")

      if [ -n "$endpoints" ]; then
        endpoints="${endpoints},"
      fi
      endpoints="${endpoints}${ip}:${ETCD_CLIENT_PORT}"
    done

    # set the endpoints in the benchmark config
    echo "Current etcd cluster endpoints: $endpoints"
    $BENCHMARK_CONTROL_BIN config set endpoints="$endpoints"

    # print the benchmark config
    echo "Current benchmark config:"
    $BENCHMARK_CONTROL_BIN config list

    # start the benchmark client service
    echo "Starting benchmark client service..."
    start_benchmark_client_service

    # run the benchmark
    $BENCHMARK_CONTROL_BIN run "$benchmark_client_pubic_ip:$BENCHMARK_CLIENT_GRPC_PORT"

    # download the benchmark client output files
    echo "Downloading benchmark client output files..."
    download_benchmark_client_output_files "$output_dir"

    # cleanup the benchmark client output files
    echo "Cleaning up benchmark client output files..."
    cleanup_benchmark_client_output_files

    # reset all etcd nodes
    echo "Resetting databases in all etcd nodes..."
    reset_all_etcd_nodes

    # restart all etcd nodes
    echo "Restarting etcd services in all etcd nodes..."
    restart_all_etcd_nodes

    echo "Sleeping for 5 minutes before running the next workload..."
    sleep 300
    # if [ "$workload" != "${WORKLOAD_TYPES[${#WORKLOAD_TYPES[@]} - 1]}" ]; then
    # fi
  done
}

main() {
  # Exit if no argument provided
  if [ $# -eq 0 ]; then
    echo "Error: No argument provided"
    print_usage
    exit 1
  fi

  while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
    # add help message using -h or --help
    -h | --help)
      print_usage
      return 0
      ;;
    -z | --zone)
      local lowercase_arg=$2
      lowercase_arg=$(echo "$2" | tr '[:upper:]' '[:lower:]')
      # Check if $2 is 'a', 'b', or 'c'
      if [[ "$lowercase_arg" =~ ^(a|b|c)$ ]]; then
        ZONE=${REGION}-${lowercase_arg}
      else
        echo "Error: the [zone] parameter must be 'a', 'b', or 'c', instead of $2"
        exit 1
      fi
      shift
      shift
      ;;
    -s | --scenario)
      if [[ "$2" != "kv" && "$2" != "lock" ]]; then
        echo "Error: the [scenario] option must be 'kv' or 'lock', instead of $2"
        exit 1
      fi
      SCENARIO="$2"
      shift
      shift
      ;;
    -n | --num_nodes)
      # check if the argument is a number and one of 1, 3, 5
      if [[ "$2" =~ ^[1|3|5]$ ]]; then
        NUM_NODES=$2
      else
        echo "Error: the [num_nodes] parameter must be 1, 3, or 5, instead of $2"
        exit 1
      fi
      shift
      shift
      ;;
    -w | --workload)
      if [ "$2" == "all" ]; then
        if [ "$SCENARIO" == "kv" ]; then
          WORKLOAD_TYPES=("${KV_WORKLOAD_TYPES[@]}")
        else
          WORKLOAD_TYPES=("${LOCK_WORKLOAD_TYPES[@]}")
        fi
      else
        # split the workloads by comma
        IFS=',' read -r -a WORKLOAD_TYPES <<<"$2"
        # check if the workloads are valid
        for workload in "${WORKLOAD_TYPES[@]}"; do
          local is_valid=false
          if [ "$SCENARIO" == "kv" ]; then
            for w in "${KV_WORKLOAD_TYPES[@]}"; do
              if [ "$workload" == "$w" ]; then
                is_valid=true
                break
              fi
            done
          else
            for w in "${LOCK_WORKLOAD_TYPES[@]}"; do
              if [ "$workload" == "$w" ]; then
                is_valid=true
                break
              fi
            done
          fi
          if [ "$is_valid" == false ]; then
            echo "Error: the [workload] parameter must be one of the following: ${KV_WORKLOAD_TYPES[*]} for scenario 'kv' or ${LOCK_WORKLOAD_TYPES[*]} for scenario 'lock', instead of $workload"
            exit 1
          fi
        done
      fi
      shift
      shift
      ;;
    -d | --out_dir)
      OUT_DIR="$2"
      shift
      shift
      ;;
    *)
      shift
      ;;
    esac
  done

  run_benchmark
}

main "$@"
