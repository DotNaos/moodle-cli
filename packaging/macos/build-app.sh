#!/usr/bin/env bash
set -euo pipefail

archive=""
version=""
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

if [[ -z "${archive}" || -z "${version}" || -z "${output}" ]]; then
  echo "Usage: $0 --archive <path> --version <tag> --output <path>" >&2
  exit 1
fi

work_dir="$(mktemp -d)"
app_dir="${work_dir}/moodle-cli.app"
contents_dir="${app_dir}/Contents"
macos_dir="${contents_dir}/MacOS"
resources_dir="${contents_dir}/Resources"
binary_dir="${resources_dir}/bin"
trap 'rm -rf "${work_dir}"' EXIT

mkdir -p "${macos_dir}" "${binary_dir}"
COPYFILE_DISABLE=1 tar -xzf "${archive}" -C "${work_dir}"
install -m 0755 "${work_dir}/moodle" "${binary_dir}/moodle"

cat > "${contents_dir}/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleDisplayName</key>
  <string>moodle-cli</string>
  <key>CFBundleExecutable</key>
  <string>moodle-cli</string>
  <key>CFBundleIdentifier</key>
  <string>com.dotnaos.moodle-cli</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>moodle-cli</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${version#v}</string>
  <key>CFBundleVersion</key>
  <string>${version#v}</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
EOF

cat > "${macos_dir}/moodle-cli" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

app_contents="$(cd "$(dirname "$0")/.." && pwd)"
app_bundle="$(dirname "${app_contents}")"
cli_bin="${app_contents}/Resources/bin/moodle"
user_bin="${HOME}/.local/bin"
link_path="${user_bin}/moodle"

if [[ "${app_bundle}" == /Volumes/* ]]; then
  /usr/bin/osascript <<'APPLESCRIPT'
display dialog "Drag moodle-cli.app into Applications first, then open it to install the command-line tool." buttons {"OK"} default button "OK" with title "moodle-cli"
APPLESCRIPT
  exit 1
fi

mkdir -p "${user_bin}"
ln -sfn "${cli_bin}" "${link_path}"

if [[ "${MOODLE_CLI_NO_TERMINAL:-}" == "1" ]]; then
  export PATH="${user_bin}:${PATH}"
  exec "${link_path}" version
fi

/usr/bin/osascript <<'APPLESCRIPT'
set launchCommand to "export PATH=\"$HOME/.local/bin:$PATH\"; clear; echo \"moodle-cli is ready.\"; echo \"The command is linked at ~/.local/bin/moodle.\"; echo; \"$HOME/.local/bin/moodle\" version; echo; echo \"You can now run moodle in this terminal.\"; exec \"$SHELL\" -l"
tell application "Terminal"
  activate
  do script launchCommand
end tell
APPLESCRIPT
EOF

chmod 0755 "${macos_dir}/moodle-cli"
find "${app_dir}" -name '._*' -delete
xattr -cr "${app_dir}"

mkdir -p "$(dirname "${output}")"
rm -rf "${output}"
cp -R "${app_dir}" "${output}"
