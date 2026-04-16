#!/usr/bin/env bash
set -euo pipefail

STACK_DIR="/opt/glycoview"
STACK_NAME="glycoview"
STACK_FILE="${STACK_DIR}/stack/stack.yml"
ENV_FILE="${STACK_DIR}/stack/.env"
ENV_EXAMPLE_FILE="${STACK_DIR}/stack/.env.example"
BOOT_ENV_FILE="/boot/firmware/glycoview-firstboot.env"

mkdir -p "${STACK_DIR}/stack"

if [[ ! -f "${ENV_FILE}" ]]; then
  if [[ -f "${BOOT_ENV_FILE}" ]]; then
    cp "${BOOT_ENV_FILE}" "${ENV_FILE}"
  elif [[ -f "${ENV_EXAMPLE_FILE}" ]]; then
    cp "${ENV_EXAMPLE_FILE}" "${ENV_FILE}"
  else
    echo "no bootstrap env file found" >&2
    exit 1
  fi
fi

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

random_secret() {
  tr -dc 'A-Za-z0-9' </dev/urandom | head -c 40
}

write_env_var() {
  local key="$1"
  local value="$2"
  if grep -qE "^${key}=" "${ENV_FILE}"; then
    perl -0pi -e 's/^'"${key}"'=.*$/'"${key}"'='"${value}"'/m' "${ENV_FILE}"
  else
    printf '\n%s=%s\n' "${key}" "${value}" >>"${ENV_FILE}"
  fi
}

if [[ -z "${POSTGRES_PASSWORD:-}" || "${POSTGRES_PASSWORD}" == "change-me" ]]; then
  POSTGRES_PASSWORD="$(random_secret)"
  write_env_var "POSTGRES_PASSWORD" "${POSTGRES_PASSWORD}"
fi

if [[ -z "${GLYCOVIEW_AGENT_TOKEN:-}" || "${GLYCOVIEW_AGENT_TOKEN}" == "replace-with-a-long-random-secret" ]]; then
  GLYCOVIEW_AGENT_TOKEN="$(random_secret)"
  write_env_var "GLYCOVIEW_AGENT_TOKEN" "${GLYCOVIEW_AGENT_TOKEN}"
fi

if [[ -z "${GLYCOVIEW_DOMAIN:-}" ]]; then
  GLYCOVIEW_DOMAIN="glycoview.local"
  write_env_var "GLYCOVIEW_DOMAIN" "${GLYCOVIEW_DOMAIN}"
fi

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

if ! docker info >/dev/null 2>&1; then
  echo "docker is not available yet" >&2
  exit 1
fi

docker compose --project-name "${STACK_NAME}" --env-file "${ENV_FILE}" -f "${STACK_FILE}" pull
docker compose --project-name "${STACK_NAME}" --env-file "${ENV_FILE}" -f "${STACK_FILE}" up -d --remove-orphans
