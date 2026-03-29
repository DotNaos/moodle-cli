#!/usr/bin/env bash
set -euo pipefail

version=""
pkg=""
output=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      version="$2"
      shift 2
      ;;
    --pkg)
      pkg="$2"
      shift 2
      ;;
    --output)
      output="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "${version}" || -z "${pkg}" || -z "${output}" ]]; then
  echo "Usage: $0 --version <tag> --pkg <path> --output <path>" >&2
  exit 1
fi

script_dir="$(cd "$(dirname "$0")" && pwd)"
work_dir="$(mktemp -d)"
stage_dir="${work_dir}/moodle-cli"
trap 'rm -rf "${work_dir}"' EXIT

mkdir -p "${stage_dir}"
cp "${pkg}" "${stage_dir}/moodle-cli.pkg"

mkdir -p "$(dirname "${output}")"
rm -f "${output}"

hdiutil create \
  -volname "moodle-cli ${version}" \
  -srcfolder "${stage_dir}" \
  -fs HFS+ \
  -format UDZO \
  "${output}"
