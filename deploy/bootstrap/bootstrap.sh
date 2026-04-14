#!/usr/bin/env bash
set -euo pipefail

STACK_DIR="/opt/bscout"
STACK_NAME="bscout"
STACK_FILE="${STACK_DIR}/stack/stack.yml"
ENV_FILE="${STACK_DIR}/stack/.env"

mkdir -p "${STACK_DIR}/stack"

if ! docker info >/dev/null 2>&1; then
  echo "docker is not available yet" >&2
  exit 1
fi

if ! docker info 2>/dev/null | grep -q "Swarm: active"; then
  docker swarm init --advertise-addr 127.0.0.1 >/dev/null 2>&1 || true
fi

if ! docker secret ls --format '{{.Name}}' | grep -qx "postgres_password"; then
  if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
    echo "POSTGRES_PASSWORD is required on first bootstrap" >&2
    exit 1
  fi
  printf '%s' "${POSTGRES_PASSWORD}" | docker secret create postgres_password - >/dev/null
fi

docker stack deploy --with-registry-auth -c "${STACK_FILE}" "${STACK_NAME}"
