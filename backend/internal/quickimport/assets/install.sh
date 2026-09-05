#!/bin/sh
set -eu
umask 077
action=${1:-}
agent=${2:-}
server=${3:-}
ticket=${4:-}
case "$action" in install|clean) ;; *) echo 'Expected install or clean' >&2; exit 1;; esac
case "$agent" in claude|codex|opencode) ;; *) echo 'Unsupported Agent' >&2; exit 1;; esac
case "$server" in https://*) ;; *) echo 'HTTPS server required' >&2; exit 1;; esac
case "$(uname -s)" in Darwin) target_os=darwin;; Linux) target_os=linux;; *) echo 'Unsupported operating system' >&2; exit 1;; esac
case "$(uname -m)" in arm64|aarch64) target_arch=arm64;; x86_64|amd64) target_arch=amd64;; *) echo 'Unsupported architecture' >&2; exit 1;; esac
protect_dir() {
  test ! -L "$1" || { echo 'Linked recovery directory refused' >&2; exit 1; }
  mkdir -p "$1"
  chmod 700 "$1"
}
test ! -L "$HOME" || { echo 'Linked user directory refused' >&2; exit 1; }
recovery="$HOME/.sub2api-quick-import"
protect_dir "$recovery"
protect_dir "$recovery/bin"
protect_dir "$recovery/$agent"
checksum_file=$(mktemp "$recovery/bin/checksum.XXXXXX")
download=$(mktemp "$recovery/bin/download.XXXXXX")
restore_temp=$(mktemp "$recovery/$agent/restore.XXXXXX")
trap 'rm -f "$checksum_file" "$download" "$restore_temp"' EXIT HUP INT TERM
asset="${server%/}/api/v1/quick-import/assets/quick-import-$target_os-$target_arch"
curl --fail --silent --show-error --proto '=https' --max-time 30 -A 'lianjieai-quick-import/2.0' "$asset.sha256" -o "$checksum_file"
checksum=$(tr -d '\r\n' < "$checksum_file")
case "$checksum" in ''|*[!a-f0-9]*) echo 'Invalid import helper checksum' >&2; exit 1;; esac
test "${#checksum}" -eq 64 || { echo 'Invalid import helper checksum' >&2; exit 1; }
binary="$recovery/bin/$checksum"
test ! -L "$binary" || { echo 'Linked import helper refused' >&2; exit 1; }
hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d ' ' -f 1
  elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | cut -d ' ' -f 1
  elif command -v openssl >/dev/null 2>&1; then openssl dgst -sha256 "$1" | sed 's/^.*= //'
  else echo 'No system SHA256 utility available' >&2; exit 1; fi
}
if test ! -f "$binary"; then
  curl --fail --silent --show-error --proto '=https' --max-time 120 -A 'lianjieai-quick-import/2.0' "$asset" -o "$download"
  test "$(hash_file "$download")" = "$checksum" || { echo 'Import helper checksum mismatch' >&2; exit 1; }
  chmod 700 "$download"
  mv "$download" "$binary"
fi
test "$(hash_file "$binary")" = "$checksum" || { echo 'Cached import helper checksum mismatch' >&2; exit 1; }
test ! -L "$recovery/$agent/restore.sh" || { echo 'Linked recovery script refused' >&2; exit 1; }
printf '#!/bin/sh\nset -eu\nexec "$(CDPATH= cd -- "$(dirname -- "$0")/../bin" && pwd)/%s" clean --agent %s\n' "$checksum" "$agent" > "$restore_temp"
chmod 700 "$restore_temp"
mv "$restore_temp" "$recovery/$agent/restore.sh"
if test "$action" = install; then
  "$binary" install --agent "$agent" --server "$server" --ticket "$ticket"
else
  "$binary" clean --agent "$agent"
fi
