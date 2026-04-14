#!/usr/bin/env bash
set -euo pipefail

if [ ! -d "${ROOTFS_DIR}" ]; then
  echo "ROOTFS_DIR is not set" >&2
  exit 1
fi

if [ ! -d "${PREV_ROOTFS_DIR}" ]; then
  echo "PREV_ROOTFS_DIR is not set" >&2
  exit 1
fi

if [ ! -e "${ROOTFS_DIR}" ]; then
  copy_previous
fi
