#!/usr/bin/env bash
set -euo pipefail

archive=""
version=""
arch=""
output=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --archive)
      archive="$2"
      shift 2
      ;;
    --version)
      version="$2"
      shift 2
      ;;
    --arch)
      arch="$2"
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

if [[ -z "${archive}" || -z "${version}" || -z "${arch}" || -z "${output}" ]]; then
  echo "Usage: $0 --archive <path> --version <tag> --arch <arch> --output <path>" >&2
  exit 1
fi

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"
work_dir="$(mktemp -d)"
stage_dir="${work_dir}/moodle-cli"
trap 'rm -rf "${work_dir}"' EXIT

mkdir -p "${stage_dir}"
tar -xzf "${archive}" -C "${work_dir}"

cp "${work_dir}/moodle" "${stage_dir}/moodle"
cp "${script_dir}/install.command" "${stage_dir}/install.command"
cp "${repo_root}/README.md" "${stage_dir}/README.md"
chmod 755 "${stage_dir}/moodle" "${stage_dir}/install.command"

cat > "${stage_dir}/INSTALL.txt" <<EOF
moodle-cli ${version} (${arch})

1. Open install.command
2. Follow the Terminal prompt
3. Run 'moodle version' to confirm the installation
EOF

mkdir -p "$(dirname "${output}")"
rm -f "${output}"

hdiutil create \
  -volname "moodle-cli ${version}" \
  -srcfolder "${stage_dir}" \
  -fs HFS+ \
  -format UDZO \
  "${output}"
