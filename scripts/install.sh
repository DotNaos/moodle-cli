#!/usr/bin/env bash
set -euo pipefail

OWNER="DotNaos"
REPO="moodle-cli"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
CHECKSUM_FILE="checksums.txt"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

download() {
  local url="$1"
  local output="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO "$output" "$url"
    return
  fi

  echo "Either curl or wget is required." >&2
  exit 1
}

sha256_file() {
  local file="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi

  echo "Either sha256sum or shasum is required." >&2
  exit 1
}

normalize_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *)
      echo "Unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

normalize_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "Unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

main() {
  require_cmd tar
  require_cmd awk
  require_cmd grep

  local os
  os="$(normalize_os)"
  local arch
  arch="$(normalize_arch)"
  local asset="moodle_${os}_${arch}.tar.gz"

  local base_url
  if [[ "$VERSION" == "latest" ]]; then
    base_url="https://github.com/${OWNER}/${REPO}/releases/latest/download"
  else
    base_url="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}"
  fi

  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT

  mkdir -p "$INSTALL_DIR"

  download "${base_url}/${asset}" "${tmp_dir}/${asset}"
  download "${base_url}/${CHECKSUM_FILE}" "${tmp_dir}/${CHECKSUM_FILE}"

  local expected
  expected="$(grep "  ${asset}$" "${tmp_dir}/${CHECKSUM_FILE}" | awk '{print $1}')"
  if [[ -z "$expected" ]]; then
    echo "Could not find checksum for ${asset}." >&2
    exit 1
  fi

  local actual
  actual="$(sha256_file "${tmp_dir}/${asset}")"
  if [[ "$expected" != "$actual" ]]; then
    echo "Checksum verification failed for ${asset}." >&2
    exit 1
  fi

  tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir"
  cp "${tmp_dir}/moodle" "${INSTALL_DIR}/moodle"
  chmod 755 "${INSTALL_DIR}/moodle"

  echo "Installed moodle to ${INSTALL_DIR}/moodle"
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      echo "Add ${INSTALL_DIR} to your PATH if it is not already there." >&2
      ;;
  esac
}

main "$@"
