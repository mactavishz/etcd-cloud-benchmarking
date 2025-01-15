#!/bin/bash

set -euo pipefail

# Usage functions
print_usage() {
  echo "Usage: $0 [command]"
  echo "Commands:"
  echo "  single  - Configure single node etcd"
  echo "  three   - Configure three node etcd cluster"
  echo "  five    - Configure five node etcd cluster"
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
REGION="europe-central2"
ZONE="${REGION}-c"
TMP_SERVICE_FILE="etcd.service"

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

# Function to configure a single node
configure_single_node() {
  local instance="etcd-single"
  local ip=$(get_internal_ip ${instance})

  # Generate systemd service file
  generate_etcd_service ${instance} "${instance}=http://${ip}:2380" "new" "${ip}"

  # Copy and enable service
  gcloud compute scp ${TMP_SERVICE_FILE} ${instance}:~/${TMP_SERVICE_FILE} --zone=${ZONE}
  rm ${TMP_SERVICE_FILE}
  gcloud compute ssh ${instance} --zone=${ZONE} --command="
        sudo mv ${TMP_SERVICE_FILE} /etc/systemd/system/
        sudo systemctl daemon-reload
        sudo systemctl enable etcd
        sudo systemctl start etcd
    "
}

# Function to configure three node cluster
configure_three_node() {
  local cluster_nodes=""
  local ips=()

  # Get IPs of all nodes
  for i in {0..2}; do
    local instance="etcd-3-${i}"
    local ip=$(get_internal_ip "${instance}")
    ips+=($ip)
    if [ -n "$cluster_nodes" ]; then
      cluster_nodes="${cluster_nodes},"
    fi
    cluster_nodes="${cluster_nodes}${instance}=http://${ip}:2380"
  done

  # Configure each node
  for i in {0..2}; do
    local instance="etcd-3-${i}"
    generate_etcd_service "${instance}" "${cluster_nodes}" "new" "${ips[$i]}"

    # Copy and enable service
    gcloud compute scp ${TMP_SERVICE_FILE} "${instance}":~/${TMP_SERVICE_FILE} --zone=${ZONE}
    rm ${TMP_SERVICE_FILE}
    gcloud compute ssh "${instance}" --zone=${ZONE} --command="
            sudo mkdir -p /var/lib/etcd
            sudo mv ${TMP_SERVICE_FILE} /etc/systemd/system/
            sudo systemctl daemon-reload
            sudo systemctl enable etcd
            sudo systemctl start etcd
        "
  done
}

# Function to configure five node cluster
configure_five_node() {
  local cluster_nodes=""
  local ips=()

  # Get IPs of all nodes
  for i in {0..4}; do
    local instance="etcd-5-${i}"
    local ip=$(get_internal_ip "${instance}")
    ips+=($ip)
    if [ -n "$cluster_nodes" ]; then
      cluster_nodes="${cluster_nodes},"
    fi
    cluster_nodes="${cluster_nodes}${instance}=http://${ip}:2380"
  done

  # Configure each node
  for i in {0..4}; do
    local instance="etcd-5-${i}"
    generate_etcd_service "${instance}" "${cluster_nodes}" "new" "${ips[$i]}"

    # Copy and enable service
    gcloud compute scp ${TMP_SERVICE_FILE} "${instance}":~/${TMP_SERVICE_FILE} --zone=${ZONE}
    rm ${TMP_SERVICE_FILE}
    gcloud compute ssh "${instance}" --zone=${ZONE} --command="
            sudo mkdir -p /var/lib/etcd
            sudo mv ${TMP_SERVICE_FILE} /etc/systemd/system/
            sudo systemctl daemon-reload
            sudo systemctl enable etcd
            sudo systemctl start etcd
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
    configure_single_node
    verify_cluster "etcd-single"
    ;;
  "three")
    configure_three_node
    verify_cluster "etcd-3-0"
    verify_cluster "etcd-3-1"
    verify_cluster "etcd-3-2"
    ;;
  "five")
    configure_five_node
    verify_cluster "etcd-5-0"
    verify_cluster "etcd-5-1"
    verify_cluster "etcd-5-2"
    verify_cluster "etcd-5-3"
    verify_cluster "etcd-5-4"
    ;;
  *)
    print_usage
    exit 1
    ;;
  esac
}

main "$@"
