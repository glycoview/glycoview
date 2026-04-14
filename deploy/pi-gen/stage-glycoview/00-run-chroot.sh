#!/usr/bin/env bash
set -euo pipefail

systemctl enable docker.service
systemctl enable avahi-daemon.service
systemctl enable glycoview-appliance-bootstrap.service
