#!/bin/bash

# Autostart services https://askubuntu.com/questions/1367139/apt-get-upgrade-auto-restart-services
export DEBIAN_FRONTEND=noninteractive

install_etcdctl() {
  # Install etcd-ctl
  ETCD_VER=v3.5.17
  DOWNLOAD_URL=https://github.com/etcd-io/etcd/releases/download
  curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o etcd-${ETCD_VER}-linux-amd64.tar.gz
  tar xzvf etcd-${ETCD_VER}-linux-amd64.tar.gz
  cd etcd-${ETCD_VER}-linux-amd64 || exit
  mv etcdctl /usr/local/bin/ && cd ..
  rm -rf etcd-${ETCD_VER}-linux-amd64
  rm -rf etcd-${ETCD_VER}-linux-amd64.tar.gz
}

install_ops_agent() {
  curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
  sudo bash add-google-cloud-ops-agent-repo.sh --also-install
}

install_golang() {
  GO_VER=1.23.4
  ARCH="amd64"
  sudo curl -O -L "https://golang.org/dl/go${GO_VER}.linux-${ARCH}.tar.gz"
  sudo rm -rf /usr/local/go
  sudo tar -xzf "go${GO_VER}.linux-${ARCH}.tar.gz"
  sudo chown -R root:root ./go
  sudo mv -v go /usr/local
  sudo rm -rf "go${GO_VER}.linux-${ARCH}.tar.gz"
  # Add to both /etc/profile and /etc/bash.bashrc for both interactive and non-interactive shells
  echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
  echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/bash.bashrc
  # Create a file in /etc/profile.d/ for system-wide PATH
  echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/golang.sh
  sudo chmod +x /etc/profile.d/golang.sh
  # Create symlink to ensure the binary is in standard PATH
  sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go
  sudo ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
}

mark_startup_finish() {
  touch /tmp/startup_completed
}

sudo apt update && sudo apt upgrade -y
sudo apt-get install -y git curl make
install_etcdctl
install_golang
install_ops_agent
mark_startup_finish
