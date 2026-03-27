#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SOURCE_BINARY="${SCRIPT_DIR}/moodle"

if [[ ! -f "${SOURCE_BINARY}" ]]; then
  echo "moodle binary not found next to the installer script."
  exit 1
fi

TARGET_DIR="${INSTALL_DIR:-/usr/local/bin}"
FALLBACK_DIR="${HOME}/.local/bin"

install_into() {
  local dir="$1"
  mkdir -p "${dir}"
  install -m 0755 "${SOURCE_BINARY}" "${dir}/moodle"
  echo
  echo "Installed moodle to ${dir}/moodle"
  if [[ ":${PATH}:" != *":${dir}:"* ]]; then
    echo "Add ${dir} to your PATH if it is not already there."
  fi
}

if [[ -w "${TARGET_DIR}" ]] || { [[ -d "${TARGET_DIR}" ]] && [[ -w "${TARGET_DIR}" ]]; }; then
  install_into "${TARGET_DIR}"
  exit 0
fi

if command -v sudo >/dev/null 2>&1; then
  echo "Installing to ${TARGET_DIR} with sudo..."
  sudo mkdir -p "${TARGET_DIR}"
  sudo install -m 0755 "${SOURCE_BINARY}" "${TARGET_DIR}/moodle"
  echo
  echo "Installed moodle to ${TARGET_DIR}/moodle"
  exit 0
fi

echo "Could not write to ${TARGET_DIR}; installing to ${FALLBACK_DIR} instead."
install_into "${FALLBACK_DIR}"
