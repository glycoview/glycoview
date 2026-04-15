#!/usr/bin/env bash
set -euo pipefail

curl -fsSL https://get.docker.com/ | CHANNEL=stable sh

systemctl enable docker.service
systemctl enable avahi-daemon.service
systemctl enable glycoview-appliance-bootstrap.service
