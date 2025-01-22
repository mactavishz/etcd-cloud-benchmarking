#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then
  set -o xtrace
fi

# Usage functions
print_usage() {
  echo "Usage: $0 <command> [options]"
  echo "Commands:"
  echo "  deploy   - Deploy etcd cluster with number of nodes specified by -n, along with 1 benchmark machine"
  echo "  cleanup  - Cleanup all resources"
  echo "Options:"
  echo "  -n, --num_nodes <number>  The number of nodes in the etcd cluster. Default: 1"
  echo "  -z  --zone <zone>         The GCP zone in which the instances are deployed, available options: a, b, c. Default: c"
  echo "  -y, --yes                 Skip confirmation prompt"
  echo "  -h, --help                Print this help message"
}

# Options
NUM_NODES=1
SKIP_PROMPT=false

# Common variables
PROJECT_ID="$(gcloud config get core/project)"
REGION="europe-central2"
ZONE="${REGION}-c"
NETWORK="etcd-network"
SUBNET="etcd-subnet"
SUBNET_RANGE="10.0.0.0/24"
TMP_SERVICE_FILE="etcd.service"

# Machine configurations
ETCD_MACHINE_TYPE="n1-standard-2"
# used for 1-node and 3-node cluster benchmarking
BENCHMARK_CLIENT_MACHINE_TYPE="n1-highcpu-8"
# used only for 5-node cluster benchmarking
BENCHMARK_CLIENT_MACHINE_TYPE_XL="n1-highcpu-16"
ETCD_DISK_SIZE="50"
BENCHMARK_DISK_SIZE="30"
IMAGE_FAMILY="ubuntu-2204-lts"
IMAGE_PROJECT="ubuntu-os-cloud"
ETCD_NODE_TAG="etcd-node"
BENCHMARK_CLIENT_TAG="benchmark-client"
STARTUP_COMPLETED_MARKER="/tmp/startup_completed"
ETCD_PD_SSD_MOUNT_POINT="/var/lib/etcd"
ETCD_DATA_DIR="/var/lib/etcd/data"

# Benchmark client configurations
BENCHMARK_CLIENT_GRPC_PORT="50051"
BENCHMARK_REPO_DIR="benchmark-repo"
GIT_REPO_URL="https://git.tu-berlin.de/mactavishz/csb-project-ws2425.git"

confirm_gcloud_project() {
  if [ "$SKIP_PROMPT" = true ]; then
    return 0
  fi
  echo "Your current project is: ${PROJECT_ID}"
  echo "Do you want to continue with this project? [y/n]"
  read -r response
  if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    return 0
  else
    echo "Please set the correct project using 'gcloud config set project PROJECT_ID' or 'gcloud init'"
    exit 1
  fi
}

# Setup network infrastructure
setup_network() {
  echo "Creating VPC network and subnets..."
  # Create VPC network
  gcloud compute networks create ${NETWORK} --subnet-mode=custom

  echo "Creating subnet..."
  # Create subnet
  gcloud compute networks subnets create ${SUBNET} \
    --network=${NETWORK} \
    --region=${REGION} \
    --range=${SUBNET_RANGE}

  echo "Creating firewall rules for internal communication..."
  # Create firewall rules for internal communication (between etcd nodes and benchmark client)
  (
    gcloud compute firewall-rules create ${NETWORK}-internal \
      --network=${NETWORK} \
      --allow=tcp,udp,icmp \
      --source-ranges=${SUBNET_RANGE}
  ) &

  echo "Creating firewall rules for etcd cluster communication..."
  (
    gcloud compute firewall-rules create ${NETWORK}-etcd-internal \
      --network=${NETWORK} \
      --allow=tcp:2379,tcp:2380 \
      --source-ranges=${SUBNET_RANGE}
  ) &

  echo "Creating firewall rules for SSH access ... (for debugging)"
  (
    gcloud compute firewall-rules create ${NETWORK}-ssh \
      --network=${NETWORK} \
      --allow=tcp:22 \
      --source-ranges=35.235.240.0/20,0.0.0.0/0 \
      --description="Allow SSH through IAP and direct access"
  ) &

  echo "Creating firewall rules for benchmark machine gRPC public access..."
  (
    gcloud compute firewall-rules create ${NETWORK}-grpc \
      --network=${NETWORK} \
      --allow=tcp:"${BENCHMARK_CLIENT_GRPC_PORT}" \
      --source-ranges=0.0.0.0/0
  ) &
  wait
}

# Function to wait for instance to be ready for SSH
wait_for_ssh() {
  local instance_name=$1
  local max_attempts=20
  local attempt=1
  local wait_time=10

  echo "Waiting for SSH to become available on ${instance_name}..."

  while [ $attempt -le $max_attempts ]; do
    if gcloud compute ssh "${instance_name}" --zone="${ZONE}" --command="echo 'SSH connection successful'" &>/dev/null; then
      echo "SSH is now available on ${instance_name}"
      return 0
    fi

    echo "Attempt ${attempt}/${max_attempts}: SSH not yet available. Waiting ${wait_time} seconds..."
    sleep $wait_time
    attempt=$((attempt + 1))
  done

  echo "Error: Failed to establish SSH connection after ${max_attempts} attempts"
  return 1
}

wait_for_startup_finish() {
  local instance_name=$1
  local max_attempts=20
  local attempt=1
  local wait_time=10

  # Wait for startup script to complete
  echo "Waiting for startup script to complete on ${instance_name}..."
  while [ $attempt -le $max_attempts ]; do
    if gcloud compute ssh "${instance_name}" --zone="${ZONE}" --command="test -f ${STARTUP_COMPLETED_MARKER}"; then
      echo "Startup script completed successfully on ${instance_name}"
      return 0
    fi
    echo "Attempt ${attempt}/${max_attempts}: startup script still not finish running, waiting for ${wait_time} seconds..."
    sleep $wait_time
    attempt=$((attempt + 1))
  done

  echo "Error: Failed to complete startup script after ${max_attempts} attempts"
  return 1
}

create_and_mount_disk() {
  local instance_name=$1
  local disk_size=$2
  local mount_point=$3
  local owner="${4:-}" # Optional owner parameter

  # Create and attach the data disk
  gcloud compute disks create "${instance_name}"-data \
    --size="${disk_size}" \
    --type=pd-ssd \
    --zone=${ZONE}

  gcloud compute instances attach-disk "${instance_name}" \
    --disk="${instance_name}"-data \
    --zone=${ZONE}

  # Wait for instance to be ready for SSH
  if ! wait_for_ssh "${name}"; then
    echo "Failed to connect to instance ${name}. Exiting..."
    exit 1
  fi

  # Format and mount command
  # See https://cloud.google.com/compute/docs/disks/format-mount-disk-linux
  local mount_commands="
    sudo mkfs.ext4 -m 0 -F -E lazy_itable_init=0,lazy_journal_init=0,discard /dev/sdb
    mount_dir=\$(eval echo ${mount_point})
    echo \${mount_dir}
    sudo mkdir -p \${mount_dir}
    echo /dev/sdb \${mount_dir} ext4 discard,defaults,nofail 0 2 | sudo tee -a /etc/fstab
    sudo mount -a"

  # Add owner change if specified
  if [[ -n "${owner}" ]]; then
    mount_commands+="
    owner=\$(eval echo ${owner})
    sudo chown -R \${owner}:\${owner} \${mount_dir}"
  fi

  # Execute commands
  gcloud compute ssh "${instance_name}" --zone=${ZONE} --command="${mount_commands}"
}

# Create etcd cluster nodes
create_etcd_node() {
  local name=$1
  local machine_type=$2

  echo "Creating etcd node, instance name: ${name}..."
  gcloud compute instances create "${name}" \
    --machine-type=${machine_type} \
    --zone=${ZONE} \
    --network=${NETWORK} \
    --subnet=${SUBNET} \
    --image-family=${IMAGE_FAMILY} \
    --image-project=${IMAGE_PROJECT} \
    --boot-disk-size=20 \
    --boot-disk-type=pd-ssd \
    --metadata-from-file startup-script=etcd_startup.sh \
    --tags=${ETCD_NODE_TAG}

  # Create and mount data disk with systemd-network owner
  create_and_mount_disk "${name}" "${ETCD_DISK_SIZE}" "${ETCD_PD_SSD_MOUNT_POINT}" "root"
}

# create multiple etcd nodes in parallel
create_etcd_nodes() {
  local node_count=$1
  local prefix=$2

  echo "Creating ${node_count} etcd nodes in parallel..."

  if [ $node_count -eq 1 ]; then
    create_etcd_node "${prefix}" "${ETCD_MACHINE_TYPE}" &
  else
    for i in $(seq 0 $((node_count - 1))); do
      create_etcd_node "${prefix}-${i}" "${ETCD_MACHINE_TYPE}" &
    done
  fi

  # Wait for all node creations to complete
  wait
}

# Create benchmark client machine
create_benchmark_machine() {
  local name=$1
  local machine_type=$2
  echo "Creating benchmark client machine, instance name: ${name}..."
  gcloud compute instances create "${name}" \
    --machine-type="${machine_type}" \
    --zone=${ZONE} \
    --network=${NETWORK} \
    --subnet=${SUBNET} \
    --image-family=${IMAGE_FAMILY} \
    --image-project=${IMAGE_PROJECT} \
    --boot-disk-size=20 \
    --boot-disk-type=pd-ssd \
    --metadata-from-file startup-script=benchmark_client_startup.sh \
    --tags=${BENCHMARK_CLIENT_TAG}

  create_and_mount_disk "${name}" "${BENCHMARK_DISK_SIZE}" "/home/\${USER}/benchmark-data" "\${USER}"

  # Wait for instance to finish startup script
  if ! wait_for_startup_finish "${name}"; then
    echo "Failed to observe startup script finishing on ${name}. Exiting..."
    exit 1
  fi

  # Clone the benchmark client repository and build the client
  gcloud compute ssh "${name}" --zone=${ZONE} --command="
  git clone ${GIT_REPO_URL} benchmark-repo
  cd ${BENCHMARK_REPO_DIR}
  make client
  sudo mv ./bin/benchclient /usr/local/bin/
  benchclient --help
  "
}

# Deploy etcd cluster with specified number of nodes
deploy_cluster() {
  local node_count=$1
  local prefix="etcd"

  # Set the appropriate instance prefix based on node count
  case $node_count in
  1) prefix="etcd-single" ;;
  3) prefix="etcd-3" ;;
  5) prefix="etcd-5" ;;
  *)
    echo "Invalid node count: $node_count"
    exit 1
    ;;
  esac

  setup_network

  # Create etcd nodes and benchmark machine in parallel
  create_etcd_nodes $node_count "$prefix" &

  # Wait for all creations to complete
  wait
}

deploy_benchmark_client() {
  local node_count=$1
  # Start benchmark machine creation in parallel
  if [ $node_count -eq 5 ]; then
    create_benchmark_machine "benchmark-client" "${BENCHMARK_CLIENT_MACHINE_TYPE_XL}" &
  else
    create_benchmark_machine "benchmark-client" "${BENCHMARK_CLIENT_MACHINE_TYPE}" &
  fi
}

parallel_cleanup() {
  # Delete instances in parallel
  echo "Deleting etcd node instances..."
  (
    gcloud compute instances list \
      --filter="tags.items=${ETCD_NODE_TAG} AND zone=${ZONE}" \
      --format="get(name)" | while read -r instance; do
      echo "Deleting instance: $instance"
      gcloud compute instances delete "$instance" --zone="${ZONE}" --quiet
      echo "Deleting data disk: ${instance}-data"
      gcloud compute disks delete "${instance}-data" --zone="${ZONE}" --quiet
    done
  ) &

  echo "Deleting benchmark instance..."
  (
    gcloud compute instances list \
      --filter="tags.items=${BENCHMARK_CLIENT_TAG} AND zone=${ZONE}" \
      --format="get(name)" | while read -r instance; do
      echo "Deleting instance: $instance"
      gcloud compute instances delete "$instance" --zone="${ZONE}" --quiet
      echo "Deleting data disk: ${instance}-data"
      gcloud compute disks delete "${instance}-data" --zone="${ZONE}" --quiet
    done

  ) &

  # Delete firewall rules in parallel
  echo "Deleting firewall rules..."
  (
    gcloud compute firewall-rules list --filter="network=${NETWORK}" \
      --format="get(name)" | while read -r rule; do
      gcloud compute firewall-rules delete "$rule" --quiet
    done
  ) &

  # Wait for all deletion operations to complete
  wait

  # Delete subnet and network sequentially as they have dependencies
  echo "Deleting subnet..."
  gcloud compute networks subnets delete "${SUBNET}" --region="${REGION}" --quiet

  echo "Deleting VPC network..."
  gcloud compute networks delete "${NETWORK}" --quiet
}

# Cleanup resources
cleanup() {
  if [ "$SKIP_PROMPT" = true ]; then
    parallel_cleanup
    return 0
  fi
  echo "Do you want to clean up all the resources in ${ZONE}? [y/n]"
  read -r response
  if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo "Cleaning up all resources..."
    parallel_cleanup &
    wait
  else
    echo "Aborting cleanup..."
    exit 1
  fi
}

# Generate etcd systemd service template
generate_etcd_service() {
  local name=$1
  local initial_cluster=$2
  local initial_cluster_state=$3
  local private_ip=$4
  local output_file=$5

  cat <<EOF >${output_file}
[Unit]
Description=etcd distributed key-value store
Documentation=https://github.com/etcd-io/etcd
After=network.target

[Service]
Type=notify
ExecStart=/usr/local/bin/etcd \\
  --name=${name} \\
  --data-dir=${ETCD_DATA_DIR} \\
  --initial-advertise-peer-urls=http://${private_ip}:2380 \\
  --listen-peer-urls=http://${private_ip}:2380 \\
  --listen-client-urls=http://${private_ip}:2379,http://127.0.0.1:2379 \\
  --advertise-client-urls=http://${private_ip}:2379 \\
  --initial-cluster-token=etcd-cluster \\
  --initial-cluster=${initial_cluster} \\
  --initial-cluster-state=${initial_cluster_state}
Restart=always
RestartSec=10s
LimitNOFILE=40000

[Install]
WantedBy=multi-user.target
EOF
}

# Function to get internal IP of an instance
get_internal_ip() {
  local instance=$1
  gcloud compute instances describe "${instance}" \
    --zone=${ZONE} \
    --format='get(networkInterfaces[0].networkIP)'
}

configure_etcd_cluster() {
  local node_count=$1
  local prefix="etcd"

  case $node_count in
  1) prefix="etcd-single" ;;
  3) prefix="etcd-3" ;;
  5) prefix="etcd-5" ;;
  *)
    echo "Error: invalid number of nodes: $node_count"
    exit 1
    ;;
  esac

  local cluster_nodes=""
  local ips=()
  local instances=()

  # Get IPs of all nodes
  for i in $(seq 0 $((node_count - 1))); do
    local instance
    if [ $node_count -eq 1 ]; then
      instance="${prefix}"
    else
      instance="${prefix}-${i}"
    fi
    instances+=("$instance")

    local ip=$(get_internal_ip "${instance}")
    ips+=($ip)

    if [ -n "$cluster_nodes" ]; then
      cluster_nodes="${cluster_nodes},"
    fi
    cluster_nodes="${cluster_nodes}${instance}=http://${ip}:2380"
  done

  # Generate and copy service files for all nodes first
  echo "Generating and copying service files..."
  for i in $(seq 0 $((node_count - 1))); do
    (
      local instance="${instances[$i]}"
      local tmp_service_file="${instance}.${TMP_SERVICE_FILE}"
      generate_etcd_service "${instance}" "${cluster_nodes}" "new" "${ips[$i]}" "${tmp_service_file}"

      # Wait for startup script to finish before copying service file
      wait_for_startup_finish "${instance}"

      # Copy service file
      echo "Copying service file to ${instance}..."
      gcloud compute scp ${tmp_service_file} "${instance}":~/${TMP_SERVICE_FILE} --zone=${ZONE}
      rm ${tmp_service_file}
    ) &
  done

  wait

  # Start all nodes in parallel
  echo "Starting etcd services on all nodes..."
  for i in $(seq 0 $((node_count - 1))); do
    (
      local instance="${instances[$i]}"
      gcloud compute ssh "${instance}" --zone=${ZONE} --command="
        sudo mkdir -p ${ETCD_DATA_DIR}
        sudo mv ${TMP_SERVICE_FILE} /etc/systemd/system/
        sudo systemctl daemon-reload
        sudo systemctl enable etcd
        sudo systemctl start etcd
      " &
    )
  done

  # Wait for all background processes to complete
  wait

  # Give the cluster some time to establish connections
  echo "Waiting for cluster to stabilize..."
  sleep 20

  # Verify cluster health for all nodes in parallel
  for i in $(seq 0 $((node_count - 1))); do
    (
      local instance="${instances[$i]}"
      echo "Verifying health of node: ${instance}"
      verify_node_health "${instance}"
      if [ "$i" -eq "$((node_count - 1))" ]; then
        # only verify cluster membership for the last node
        verify_cluster_membership "${instance}"
      fi
    ) &
  done

  # Wait for all background processes to complete
  wait
}

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

start_deploy() {
  echo "Starting deployment..."
  echo "Deploying etcd cluster with ${NUM_NODES} nodes..."
  echo "Zone: ${ZONE}"
  case "$NUM_NODES" in
  1)
    confirm_gcloud_project
    deploy_cluster 1
    deploy_benchmark_client 1
    configure_etcd_cluster 1
    ;;
  3)
    confirm_gcloud_project
    deploy_cluster 3
    deploy_benchmark_client 3
    configure_etcd_cluster 3
    ;;
  5)
    confirm_gcloud_project
    deploy_cluster 5
    deploy_benchmark_client 5
    configure_etcd_cluster 5
    ;;
  *)
    print_usage
    exit 1
    ;;
  esac
}

main() {
  # Exit if no argument provided
  local cmd=""
  if [ $# -eq 0 ]; then
    echo "Error: No argument provided"
    print_usage
    exit 1
  fi

  while [[ $# -gt 0 ]]; do
    case "$1" in
    # add help message using -h or --help
    -h | --help)
      print_usage
      return 0
      ;;
    -y | --yes)
      SKIP_PROMPT=true
      shift
      ;;
    -z | --zone)
      local lowercase_arg=$2
      lowercase_arg=$(echo "$2" | tr '[:upper:]' '[:lower:]')
      # Check if $2 is 'a', 'b', or 'c'
      if [[ "$lowercase_arg" =~ ^(a|b|c)$ ]]; then
        ZONE=${REGION}-${lowercase_arg}
        echo "Setting zone to ${ZONE}"
      else
        echo "Error: the [zone] option must be 'a', 'b', or 'c', instead of $2"
        exit 1
      fi
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
    -*)
      echo "Unknown option: $1"
      exit 1
      ;;
    *)
      cmd="$1"
      # arg must be exactly one of 'deploy', 'cleanup'
      if [[ "$cmd" != "deploy" && "$cmd" != "cleanup" ]]; then
        echo "Error: the [command] parameter must be 'deploy' or 'cleanup', instead of $1"
        exit 1
      fi
      shift # Remove the argument
      ;;
    esac
  done

  if [[ "$cmd" == "deploy" ]]; then
    start_deploy
  elif [[ "$cmd" == "cleanup" ]]; then
    cleanup
  fi
}

main "$@"
