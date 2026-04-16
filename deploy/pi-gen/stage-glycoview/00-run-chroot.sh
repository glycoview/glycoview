#!/usr/bin/env bash
set -euo pipefail

install -m 0755 -d /etc/apt/keyrings
curl -fsSL "https://download.docker.com/linux/debian/gpg" -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=arm64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian trixie stable" >/etc/apt/sources.list.d/docker.list

apt-get update

docker_version="$(apt-cache madison docker-ce | awk '/5:28\\./ { print $3; exit }')"
if [[ -z "${docker_version}" ]]; then
  echo "could not find a Docker 28.x package in the Docker apt repository" >&2
  exit 1
fi

DEBIAN_FRONTEND=noninteractive apt-get install -y \
  "docker-ce=${docker_version}" \
  "docker-ce-cli=${docker_version}" \
  containerd.io \
  docker-buildx-plugin \
  docker-compose-plugin

apt-mark hold docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin >/dev/null

systemctl enable docker.service
systemctl enable avahi-daemon.service
systemctl enable glycoview-appliance-bootstrap.service
