#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PI_GEN_DIR="${ROOT_DIR}/.tmp/pi-gen"
OUT_DIR="${ROOT_DIR}/out/pi-gen"
WORK_DIR="${ROOT_DIR}/out/pi-gen-work"

RELEASE_TAG="${1:?release tag is required}"
IMAGE_NAME="${2:?image name is required}"
FIRST_USER_PASS="${3:?first user password is required}"

mkdir -p "${ROOT_DIR}/.tmp" "${OUT_DIR}" "${WORK_DIR}"
rm -rf "${PI_GEN_DIR}"

sudo apt-get update
sudo apt-get install -y --no-install-recommends \
  coreutils quilt parted debootstrap zerofree zip dosfstools e2fsprogs \
  libarchive-tools libcap2-bin grep rsync xz-utils file git curl bc gpg pigz xxd \
  arch-test bmap-tools kmod debian-archive-keyring

git clone --depth 1 --branch arm64 https://github.com/RPi-Distro/pi-gen.git "${PI_GEN_DIR}"

cat >"${PI_GEN_DIR}/config.glycoview" <<EOF
IMG_NAME='${IMAGE_NAME}'
PI_GEN_RELEASE='Raspberry Pi reference'
RELEASE='bookworm'
DEPLOY_COMPRESSION='xz'
COMPRESSION_LEVEL='6'
LOCALE_DEFAULT='en_US.UTF-8'
TARGET_HOSTNAME='glycoview'
KEYBOARD_KEYMAP='us'
KEYBOARD_LAYOUT='English (US)'
TIMEZONE_DEFAULT='Europe/Berlin'
FIRST_USER_NAME='glycoview'
FIRST_USER_PASS='${FIRST_USER_PASS}'
ENABLE_SSH='0'
ENABLE_CLOUD_INIT='0'
EXPORT_LAST_STAGE_ONLY='1'
STAGE_LIST='stage0 stage1 stage2 ${ROOT_DIR}/deploy/pi-gen/stage-glycoview'
WORK_DIR='${WORK_DIR}'
DEPLOY_DIR='${OUT_DIR}'
EOF

cd "${PI_GEN_DIR}"
sudo ./build.sh -c config.glycoview
