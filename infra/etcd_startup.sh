#!/bin/bash
# Autostart services https://askubuntu.com/questions/1367139/apt-get-upgrade-auto-restart-services
export DEBIAN_FRONTEND=noninteractive

install_etcd() {
  # Install etcd
  local ETCD_VER=v3.5.17
  local DOWNLOAD_URL=https://github.com/etcd-io/etcd/releases/download
  curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o etcd-${ETCD_VER}-linux-amd64.tar.gz
  tar xzvf etcd-${ETCD_VER}-linux-amd64.tar.gz
  cd etcd-${ETCD_VER}-linux-amd64 || exit
  mv etcd etcdctl /usr/local/bin/ && cd ..
  rm -rf etcd-${ETCD_VER}-linux-amd64
  rm -rf etcd-${ETCD_VER}-linux-amd64.tar.gz
}

install_ops_agent() {
  curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
  sudo bash add-google-cloud-ops-agent-repo.sh --also-install
}

sudo apt update && sudo apt upgrade && sudo apt-get install -y git curl
install_etcd
install_ops_agent
