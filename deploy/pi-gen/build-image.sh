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

KEYRING_PATH="${PI_GEN_DIR}/debian-archive-keyring.gpg"
rm -f "${KEYRING_PATH}"
for key_url in \
  https://ftp-master.debian.org/keys/archive-key-12.asc \
  https://ftp-master.debian.org/keys/archive-key-12-security.asc \
  https://ftp-master.debian.org/keys/release-12.asc \
  https://ftp-master.debian.org/keys/archive-key-13.asc \
  https://ftp-master.debian.org/keys/archive-key-13-security.asc \
  https://ftp-master.debian.org/keys/release-13.asc
do
  curl -fsSL "${key_url}" | gpg --dearmor >> "${KEYRING_PATH}"
done

python3 - "${PI_GEN_DIR}" <<'PY'
from pathlib import Path
import sys

base = Path(sys.argv[1])
common = base / "scripts" / "common"
text = common.read_text()
old = '#BOOTSTRAP_ARGS+=(--keyring "${STAGE_DIR}/files/raspberrypi.gpg")'
new = 'BOOTSTRAP_ARGS+=(--keyring "${BASE_DIR}/debian-archive-keyring.gpg")'
if old not in text:
    raise SystemExit("expected bootstrap keyring placeholder not found")
common.write_text(text.replace(old, new))

apt_run = base / "stage0" / "00-configure-apt" / "00-run.sh"
text = apt_run.read_text()
needle = 'install -m 644 files/raspberrypi-archive-keyring.pgp "${ROOTFS_DIR}/usr/share/keyrings/"\n'
replacement = needle + 'install -m 644 "${BASE_DIR}/debian-archive-keyring.gpg" "${ROOTFS_DIR}/usr/share/keyrings/debian-archive-keyring.pgp"\n'
if needle not in text:
    raise SystemExit("expected apt keyring install line not found")
apt_run.write_text(text.replace(needle, replacement, 1))
PY

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
