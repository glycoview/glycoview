#!/usr/bin/env bash
set -euo pipefail

install -d "${ROOTFS_DIR}"
cp -a files/. "${ROOTFS_DIR}/"
