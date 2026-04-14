#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
STAGE_DIR="${ROOT_DIR}/deploy/pi-gen/stage-glycoview"
SUBSTAGE_DIR="${STAGE_DIR}/01-glycoview"
FILES_DIR="${SUBSTAGE_DIR}/files"
STACK_DIR="${FILES_DIR}/opt/glycoview/stack"
BOOTSTRAP_DIR="${FILES_DIR}/opt/glycoview/bootstrap"
SYSTEMD_DIR="${FILES_DIR}/etc/systemd/system"

RELEASE_TAG="${1:-latest}"

rm -rf "${SUBSTAGE_DIR}"
mkdir -p "${STACK_DIR}" "${BOOTSTRAP_DIR}" "${SYSTEMD_DIR}"

cp "${ROOT_DIR}/deploy/swarm/stack.yml" "${STACK_DIR}/stack.yml"
cp "${ROOT_DIR}/deploy/bootstrap/bootstrap.sh" "${BOOTSTRAP_DIR}/bootstrap.sh"
cp "${ROOT_DIR}/deploy/bootstrap/glycoview-appliance-bootstrap.service" "${SYSTEMD_DIR}/glycoview-appliance-bootstrap.service"
cp "${ROOT_DIR}/deploy/bootstrap/firstboot.env.example" "${STACK_DIR}/.env.example"

chmod +x "${BOOTSTRAP_DIR}/bootstrap.sh"
cp "${STAGE_DIR}/00-packages" "${SUBSTAGE_DIR}/00-packages"
cp "${STAGE_DIR}/00-run-chroot.sh" "${SUBSTAGE_DIR}/00-run-chroot.sh"
chmod +x "${SUBSTAGE_DIR}/00-run-chroot.sh"

for path in "${STACK_DIR}/.env.example" "${BOOT_DIR}/glycoview-firstboot.env"; do
  perl -0pi -e 's/^GLYCOVIEW_TAG=.*/GLYCOVIEW_TAG='"${RELEASE_TAG}"'/m; s/^GLYCOVIEW_AGENT_TAG=.*/GLYCOVIEW_AGENT_TAG='"${RELEASE_TAG}"'/m' "${path}"
done
