#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then
  set -o xtrace
fi

# Usage functions
print_usage() {
  echo "Usage: $0 [command] [zone]"
  echo "Commands:"
  echo "  single  - Deploy single node etcd cluster with benchmark machine"
  echo "  three   - Deploy three node etcd cluster with benchmark machine"
  echo "  five    - Deploy five node etcd cluster with benchmark machine"
  echo "  cleanup - Cleanup all resources"
  echo "Arguments:"
  echo "  zone    - Zone, in which the resources are created and deployed, use only 'a', 'b', or 'c'"
}

# Exit if no argument provided
if [ $# -eq 0 ]; then
  echo "Error: No argument provided"
  print_usage
  exit 1
fi

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
BENCHMARK_CLIENT_MACHINE_TYPE="n1-standard-4"
ETCD_DISK_SIZE="50"
BENCHMARK_DISK_SIZE="30"
IMAGE_FAMILY="ubuntu-2204-lts"
IMAGE_PROJECT="ubuntu-os-cloud"
ETCD_NODE_TAG="etcd-node"
BENCHMARK_CLIENT_TAG="benchmark-client"
STARTUP_COMPLETED_MARKER="/tmp/startup_completed"

# Benchmark client configurations
BENCHMARK_CLIENT_GRPC_PORT="50051"
GIT_REPO_URL="https://git.tu-berlin.de/mactavishz/csb-project-ws2425"

confirm_gcloud_project() {
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
  gcloud compute firewall-rules create ${NETWORK}-internal \
    --network=${NETWORK} \
    --allow=tcp,udp,icmp \
    --source-ranges=${SUBNET_RANGE}

  echo "Creating firewall rules for etcd cluster communication..."
  gcloud compute firewall-rules create ${NETWORK}-etcd-internal \
    --network=${NETWORK} \
    --allow=tcp:2379,tcp:2380 \
    --source-ranges=${SUBNET_RANGE}

  echo "Creating firewall rules for SSH access ... (for debugging)"
  gcloud compute firewall-rules create ${NETWORK}-ssh \
    --network=${NETWORK} \
    --allow=tcp:22 \
    --source-ranges=35.235.240.0/20,0.0.0.0/0 \
    --description="Allow SSH through IAP and direct access"

  echo "Creating firewall rules for benchmark machine gRPC public access..."
  gcloud compute firewall-rules create ${NETWORK}-grpc \
    --network=${NETWORK} \
    --allow=tcp:"${BENCHMARK_CLIENT_GRPC_PORT}" \
    --source-ranges=0.0.0.0/0
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
    sudo chown -R ${owner}:${owner} ${mount_point}"
  fi

  # Execute commands
  gcloud compute ssh "${instance_name}" --zone=${ZONE} --command="${mount_commands}"
}

# Create etcd cluster nodes
create_etcd_node() {
  local name=$1
  local index=$2

  echo "Creating etcd node, instance name: ${name}..."
  gcloud compute instances create "${name}" \
    --machine-type=${ETCD_MACHINE_TYPE} \
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
  create_and_mount_disk "${name}" "${ETCD_DISK_SIZE}" "/var/lib/etcd" "systemd-network"
}

# Create benchmark client machine
create_benchmark_machine() {
  local name=$1
  gcloud compute instances create "${name}" \
    --machine-type="${BENCHMARK_CLIENT_MACHINE_TYPE}" \
    --zone=${ZONE} \
    --network=${NETWORK} \
    --subnet=${SUBNET} \
    --image-family=${IMAGE_FAMILY} \
    --image-project=${IMAGE_PROJECT} \
    --boot-disk-size=20 \
    --boot-disk-type=pd-ssd \
    --metadata-from-file startup-script=benchmark_client_startup.sh \
    --tags=${BENCHMARK_CLIENT_TAG}

  create_and_mount_disk "${name}" "${BENCHMARK_DISK_SIZE}" "/home/\${USER}/benchmark-data"

  # Wait for instance to finish startup script
  if ! wait_for_startup_finish "${name}"; then
    echo "Failed to observe startup script finishing on ${name}. Exiting..."
    exit 1
  fi

  # Clone the benchmark client repository and build the client
  gcloud compute ssh "${name}" --zone=${ZONE} --command="
  git clone ${GIT_REPO_URL} benchmark-repo
  cd benchmark-repo
  make client
  sudo mv benchclient /usr/local/bin/
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

  # Create etcd nodes
  if [ $node_count -eq 1 ]; then
    create_etcd_node "${prefix}" 0
  else
    for i in $(seq 0 $((node_count - 1))); do
      create_etcd_node "${prefix}-${i}" "$i"
    done
  fi

  create_benchmark_machine "benchmark-client"
}

# Cleanup resources
cleanup() {
  echo "Do you want to clean up all the resources? [y/n]"
  read -r response
  if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo "Cleaning up all resources..."
  else
    echo "Aborting cleanup..."
    exit 1
  fi

  echo "Deleting etcd node instances with ${ETCD_NODE_TAG} tag..."
  gcloud compute instances list \
    --filter="tags.items=${ETCD_NODE_TAG} AND zone=${ZONE}" \
    --format="get(name)" |
    while read -r instance; do
      echo "Deleting instance: $instance"
      gcloud compute instances delete "$instance" --zone="${ZONE}" --quiet
      echo "Deleting data disk: ${instance}-data"
      gcloud compute disks delete "${instance}-data" --zone="${ZONE}" --quiet
    done

  echo "Deleting benchmark instance..."
  gcloud compute instances list \
    --filter="tags.items=${BENCHMARK_CLIENT_TAG} AND zone=${ZONE}" \
    --format="get(name)" |
    while read -r instance; do
      echo "Deleting instance: $instance"
      gcloud compute instances delete "$instance" --zone="${ZONE}" --quiet
      echo "Deleting data disk: ${instance}-data"
      gcloud compute disks delete "${instance}-data" --zone="${ZONE}" --quiet
    done

  echo "Deleting remaining disks in zone ${ZONE}..."
  gcloud compute disks list \
    --filter="zone:${ZONE}" \
    --format="get(name)" |
    while read -r disk; do
      if [[ $disk == etcd-* ]] || [[ $disk == benchmark-* ]]; then
        echo "Deleting disk: $disk"
        gcloud compute disks delete "$disk" --zone="${ZONE}" --quiet
      fi
    done

  echo "Deleting firewall rules..."
  gcloud compute firewall-rules list --filter="network=${NETWORK}" --format="get(name)" |
    while read -r rule; do
      gcloud compute firewall-rules delete "$rule" --quiet
    done

  echo "Deleting subnet..."
  gcloud compute networks subnets delete "${SUBNET}" --region="${REGION}" --quiet

  echo "Deleting VPC network..."
  gcloud compute networks delete "${NETWORK}" --quiet
}

# Generate etcd systemd service template
generate_etcd_service() {
  local name=$1
  local initial_cluster=$2
  local initial_cluster_state=$3
  local private_ip=$4

  cat <<EOF >${TMP_SERVICE_FILE}
[Unit]
Description=etcd distributed key-value store
Documentation=https://github.com/etcd-io/etcd
After=network.target

[Service]
Type=notify
ExecStart=/usr/local/bin/etcd \\
  --name=${name} \\
  --data-dir=/var/lib/etcd \\
  --listen-peer-urls=http://${private_ip}:2380 \\
  --listen-client-urls=http://${private_ip}:2379,http://127.0.0.1:2379 \\
  --initial-advertise-peer-urls=http://${private_ip}:2380 \\
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

  # Get IPs of all nodes
  for i in $(seq 0 $((node_count - 1))); do
    local instance
    if [ $node_count -eq 1 ]; then
      instance="${prefix}"
    else
      instance="${prefix}-${i}"
    fi

    local ip=$(get_internal_ip "${instance}")
    ips+=($ip)

    if [ -n "$cluster_nodes" ]; then
      cluster_nodes="${cluster_nodes},"
    fi
    cluster_nodes="${cluster_nodes}${instance}=http://${ip}:2380"
  done

  for i in $(seq 0 $((node_count - 1))); do
    local instance
    if [ $node_count -eq 1 ]; then
      instance="${prefix}"
    else
      instance="${prefix}-${i}"
    fi

    generate_etcd_service "${instance}" "${cluster_nodes}" "new" "${ips[$i]}"
    wait_for_startup_finish "${instance}"

    # Copy and enable service
    gcloud compute scp ${TMP_SERVICE_FILE} "${instance}":~/${TMP_SERVICE_FILE} --zone=${ZONE}
    rm ${TMP_SERVICE_FILE}
    gcloud compute ssh "${instance}" --zone=${ZONE} --command="
            sudo mv ${TMP_SERVICE_FILE} /etc/systemd/system/
            sudo systemctl daemon-reload
            sudo systemctl enable etcd
            sudo systemctl start etcd
        "
  done

  # Verify cluster health for all nodes
  for i in $(seq 0 $((node_count - 1))); do
    local instance
    if [ $node_count -eq 1 ]; then
      instance="${prefix}"
    else
      instance="${prefix}-${i}"
    fi

    echo "Verifying health of node: ${instance}"
    gcloud compute ssh "${instance}" --zone=${ZONE} --command="
            ETCDCTL_API=3 etcdctl endpoint health --cluster
            ETCDCTL_API=3 etcdctl member list
        "
  done
}

# Function to verify cluster health
verify_cluster() {
  local instance=$1
  gcloud compute ssh "${instance}" --zone=${ZONE} --command="
        ETCDCTL_API=3 etcdctl endpoint health --cluster
        ETCDCTL_API=3 etcdctl member list
    "
}

main() {
  # Main script execution
  if [ $# -eq 2 ]; then
    LOWERCASE_ARG=$(echo "$2" | tr '[:upper:]' '[:lower:]')
    # Check if $2 is 'a', 'b', or 'c'
    if [[ "$LOWERCASE_ARG" =~ ^(a|b|c)$ ]]; then
      ZONE=${REGION}-${LOWERCASE_ARG}
      echo "Use custom zone: ${ZONE}"
    else
      echo "Error: the [zone] parameter must be 'a', 'b', or 'c', instead of $2"
      exit 1
    fi
  elif [ $# -gt 2 ]; then
    echo "Error: too much parameters"
  else
    echo "Use default zone: ${ZONE}"
  fi

  case "$1" in
  "single")
    confirm_gcloud_project
    deploy_cluster 1
    configure_etcd_cluster 1
    ;;
  "three")
    confirm_gcloud_project
    deploy_cluster 3
    configure_etcd_cluster 3
    ;;
  "five")
    confirm_gcloud_project
    deploy_cluster 5
    configure_etcd_cluster 5
    ;;
  "cleanup")
    cleanup
    ;;
  *)
    print_usage
    exit 1
    ;;
  esac
}

main "$@"
