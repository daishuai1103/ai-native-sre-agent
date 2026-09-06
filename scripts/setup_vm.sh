#!/bin/bash
set -e

echo "==> Updating packages"
sudo apt update && sudo apt upgrade -y

echo "==> Installing base tools"
sudo apt install -y git curl wget make ca-certificates gnupg lsb-release

echo "==> Installing Docker"
if ! command -v docker &>/dev/null; then
  curl -fsSL https://get.docker.com | sh
  sudo usermod -aG docker "$USER"
else
  echo "Docker already installed, skipping."
fi

echo "==> Installing Docker Compose plugin"
sudo apt install -y docker-compose-plugin

echo "==> Installing Go 1.22"
GO_VERSION=1.22.0
if ! command -v go &>/dev/null; then
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
  rm -f "go${GO_VERSION}.linux-amd64.tar.gz"
  if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
  fi
  export PATH=$PATH:/usr/local/go/bin
else
  echo "Go already installed: $(go version)"
fi

echo "==> Installing Node.js 20"
if ! command -v node &>/dev/null; then
  curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
  sudo apt install -y nodejs
else
  echo "Node.js already installed: $(node -v)"
fi

echo "==> Installing Python tools"
sudo apt install -y python3 python3-pip python3-venv

echo "==> Done. Please log out and back in for Docker group changes to take effect."
echo "==> After re-login, verify with: docker ps && go version && node -v"
