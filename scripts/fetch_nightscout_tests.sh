#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REF="${1:-9cd304f78a5c12401c9711cbd56d2a12eaca0632}"
REPO_URL="https://github.com/nightscout/cgm-remote-monitor.git"
WORK_DIR="$(mktemp -d)"
DEST_DIR="${ROOT_DIR}/third_party/nightscout"

cleanup() {
  rm -rf "${WORK_DIR}"
}

trap cleanup EXIT

git clone --filter=blob:none --quiet "${REPO_URL}" "${WORK_DIR}/repo"
git -C "${WORK_DIR}/repo" checkout --quiet "${REF}"

rm -rf "${DEST_DIR}"
mkdir -p "${DEST_DIR}"

cp "${WORK_DIR}/repo/LICENSE" "${DEST_DIR}/LICENSE"
cp "${WORK_DIR}/repo/COPYRIGHT" "${DEST_DIR}/COPYRIGHT"
cp "${WORK_DIR}/repo/README.md" "${DEST_DIR}/README.upstream.md"
cp "${WORK_DIR}/repo/package.json" "${DEST_DIR}/package.json"

mkdir -p "${DEST_DIR}/lib/api3"
cp "${WORK_DIR}/repo/lib/api3/swagger.yaml" "${DEST_DIR}/lib/api3/swagger.yaml"
cp "${WORK_DIR}/repo/lib/api3/swagger.json" "${DEST_DIR}/lib/api3/swagger.json"

cp -R "${WORK_DIR}/repo/tests" "${DEST_DIR}/tests"
cp -R "${WORK_DIR}/repo/testing" "${DEST_DIR}/testing"

COMMIT_DATE="$(git -C "${WORK_DIR}/repo" log -1 --date=iso --format='%cd')"

cat > "${DEST_DIR}/README.md" <<EOF
# Nightscout Upstream Snapshot

This directory contains imported test and contract artifacts from:

- repository: ${REPO_URL}
- commit: ${REF}
- commit date: ${COMMIT_DATE}

Imported content:

- \`tests/\`
- \`testing/\`
- \`lib/api3/swagger.yaml\`
- \`lib/api3/swagger.json\`
- upstream \`LICENSE\`, \`COPYRIGHT\`, \`README.md\`, and \`package.json\`

These files remain attributed to the Nightscout project and are used here as compatibility references for the Go/Postgres port.
EOF

printf '{\n  "repository": "%s",\n  "commit": "%s",\n  "commit_date": "%s"\n}\n' \
  "${REPO_URL}" "${REF}" "${COMMIT_DATE}" > "${DEST_DIR}/UPSTREAM_SNAPSHOT.json"

echo "Imported Nightscout snapshot ${REF} into ${DEST_DIR}"
