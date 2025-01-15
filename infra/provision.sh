#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then
    set -o xtrace
fi

# Usage functions
print_usage() {
    echo "Usage: $0 [command]"
    echo "Commands:"
    echo "  single  - Deploy single node etcd cluster with benchmark machine"
    echo "  three   - Deploy three node etcd cluster with benchmark machine"
    echo "  five    - Deploy five node etcd cluster with benchmark machine"
    echo "  cleanup - Cleanup all resources"
}

# Exit if no argument provided
if [ $# -eq 0 ]; then
    echo "Error: No argument provided"
    print_usage
    exit 1
fi


# Common variables
PROJECT_ID="$(gcloud config get core/project)"
REGION="europe-west3"
ZONE="${REGION}-c"
NETWORK="etcd-network"
SUBNET="etcd-subnet"
SUBNET_RANGE="10.0.0.0/24"

# Machine configurations
ETCD_MACHINE_TYPE="n1-standard-2"
BENCHMARK_CLIENT_MACHINE_TYPE="n1-standard-4"
ETCD_DISK_SIZE="50"
BENCHMARK_DISK_SIZE="30"
IMAGE_FAMILY="ubuntu-2204-lts"
IMAGE_PROJECT="ubuntu-os-cloud"
ETCD_NODE_TAG="etcd-node"
BENCHMARK_CLIENT_TAG="benchmark-client"

# Benchmark client configurations
BENCHMARK_CLIENT_GRPC_PORT="50051"

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

    # Create and attach the data disk
    # See https://cloud.google.com/compute/docs/disks/add-persistent-disk
    gcloud compute disks create "${name}"-data \
        --size=${ETCD_DISK_SIZE} \
        --type=pd-ssd \
        --zone=${ZONE}

    gcloud compute instances attach-disk "${name}" \
        --disk="${name}"-data \
        --zone=${ZONE}

    # Wait for instance to be ready for SSH
    if ! wait_for_ssh "${name}"; then
        echo "Failed to connect to instance ${name}. Exiting..."
        exit 1
    fi

    # Format and mount the data disk
    # See https://cloud.google.com/compute/docs/disks/format-mount-disk-linux
    gcloud compute ssh "${name}" --zone=${ZONE} --command="
        sudo mkfs.ext4 -m 0 -F -E lazy_itable_init=0,lazy_journal_init=0,discard /dev/sdb
        sudo mkdir -p /var/lib/etcd
        echo '/dev/sdb /var/lib/etcd ext4 discard,defaults,nofail 0 2' | sudo tee -a /etc/fstab
        sudo mount -a
        sudo chown -R systemd-network:systemd-network /var/lib/etcd"
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


    # Create and attach the data disk
    # See https://cloud.google.com/compute/docs/disks/add-persistent-disk
    gcloud compute disks create "${name}"-data \
        --size=${BENCHMARK_DISK_SIZE} \
        --type=pd-ssd \
        --zone=${ZONE}

    gcloud compute instances attach-disk "${name}" \
        --disk="${name}"-data \
        --zone=${ZONE}

    # Wait for instance to be ready for SSH
    if ! wait_for_ssh "${name}"; then
        echo "Failed to connect to instance ${name}. Exiting..."
        exit 1
    fi

    # Format and mount the data disk
    # See https://cloud.google.com/compute/docs/disks/format-mount-disk-linux
    gcloud compute ssh "${name}" --zone=${ZONE} --command="
        sudo mkfs.ext4 -m 0 -F -E lazy_itable_init=0,lazy_journal_init=0,discard /dev/sdb
        mkdir -p /home/${USER}/benchmark-data
        echo '/dev/sdb /home/${USER}/benchmark-data ext4 discard,defaults,nofail 0 2' | sudo tee -a /etc/fstab
        sudo mount -a
        "
}

# Deploy single node etcd cluster
deploy_single_node() {
    setup_network
    create_etcd_node "etcd-single" 0
    create_benchmark_machine "benchmark-client"
}

# Deploy 3-node etcd cluster
deploy_three_node() {
    setup_network
    for i in {0..2}; do
        create_etcd_node "etcd-3-${i}" "$i"
    done
    create_benchmark_machine "benchmark-client"
}

# Deploy 5-node etcd cluster
deploy_five_node() {
    setup_network
    for i in {0..4}; do
        create_etcd_node "etcd-5-${i}" "$i"
    done
    create_benchmark_machine "benchmark-client"
}

# Function to wait for instance to be ready for SSH
wait_for_ssh() {
    local instance_name=$1
    local max_attempts=30
    local attempt=1
    local wait_time=10

    echo "Waiting for SSH to become available on ${instance_name}..."

    while [ $attempt -le $max_attempts ]; do
        if gcloud compute ssh "${instance_name}" --zone="${ZONE}" --command="echo 'SSH connection successful'" &> /dev/null; then
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
        --format="get(name)" | \
    while read -r instance; do
        echo "Deleting instance: $instance"
        gcloud compute instances delete "$instance" --zone="${ZONE}" --quiet
        echo "Deleting data disk: ${instance}-data"
        gcloud compute disks delete "${instance}-data" --zone="${ZONE}" --quiet
    done

    echo "Deleting benchmark instance..."
    gcloud compute instances list \
        --filter="tags.items=${BENCHMARK_CLIENT_TAG} AND zone=${ZONE}" \
        --format="get(name)" | \
    while read -r instance; do
        echo "Deleting instance: $instance"
        gcloud compute instances delete "$instance" --zone="${ZONE}" --quiet
        echo "Deleting data disk: ${instance}-data"
        gcloud compute disks delete "${instance}-data" --zone="${ZONE}" --quiet
    done

    echo "Deleting remaining disks in zone ${ZONE}..."
    gcloud compute disks list \
        --filter="zone:${ZONE}" \
        --format="get(name)" | \
    while read -r disk; do
        if [[ $disk == etcd-* ]] || [[ $disk == benchmark-* ]]; then
            echo "Deleting disk: $disk"
            gcloud compute disks delete "$disk" --zone="${ZONE}" --quiet
        fi
    done

    echo "Deleting firewall rules..."
    gcloud compute firewall-rules list --filter="network=${NETWORK}" --format="get(name)" | \
    while read -r rule; do
        gcloud compute firewall-rules delete "$rule" --quiet
    done

    echo "Deleting subnet..."
    gcloud compute networks subnets delete "${SUBNET}" --region="${REGION}" --quiet

    echo "Deleting VPC network..."
    gcloud compute networks delete "${NETWORK}" --quiet
}

main() {
# Main script execution
case "$1" in
    "single")
        confirm_gcloud_project
        deploy_single_node
        ;;
    "three")
        confirm_gcloud_project
        deploy_three_node
        ;;
    "five")
        confirm_gcloud_project
        deploy_five_node
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

main "$1"
