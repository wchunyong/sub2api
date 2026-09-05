#!/bin/sh
# Run in Linux/macOS with the native binary and source directory as arguments.
set -eu
binary=$1
source_dir=$2
root=$(mktemp -d /tmp/lianjieai-native.XXXXXX)
case "$root" in /tmp/lianjieai-native.*) ;; *) exit 1;; esac
trap 'rm -rf "$root"' EXIT HUP INT TERM
export HOME="$root"
for agent in claude codex opencode; do
  printf '{"version":1,"agent":"%s","api_key":"mock-native-key","base_url":"https://example.com/v1","probe_url":"https://example.com/v1/models","model":"mock-model","protocol":"openai"}' "$agent" | "$binary" install --agent "$agent" --stdin --skip-client-check --home "$root"
  "$binary" clean --agent "$agent" --home "$root" --yes
done
# Exercise actual downloader/cache/checksum/offline scripts using local fixture transport.
mkdir "$root/mockbin"
cat > "$root/mockbin/curl" <<'CURL'
#!/bin/sh
set -eu
url=
out=
while [ "$#" -gt 0 ]; do
 case "$1" in -o) out=$2; shift;; https://*) url=$1;; esac
 shift
done
cp "$SOURCE_ASSETS/${url##*/}" "$out"
CURL
chmod +x "$root/mockbin/curl"
export SOURCE_ASSETS="$source_dir/native-assets"
export PATH="$root/mockbin:/bin:/usr/bin"
printf '{"version":1,"agent":"opencode","api_key":"mock-native-key","base_url":"https://example.com/v1","probe_url":"https://example.com/v1/models","model":"mock-model","protocol":"openai"}' | "$binary" install --agent opencode --stdin --skip-client-check --home "$root"
printf 'y\n' | sh "$source_dir/assets/install.sh" clean opencode https://example.com
[ ! -f "$root/.config/opencode/opencode.json" ]
printf '{"version":1,"agent":"opencode","api_key":"mock-native-key","base_url":"https://example.com/v1","probe_url":"https://example.com/v1/models","model":"mock-model","protocol":"openai"}' | "$binary" install --agent opencode --stdin --skip-client-check --home "$root"
printf 'y\n' | sh "$root/.sub2api-quick-import/opencode/restore.sh"
[ ! -f "$root/.config/opencode/opencode.json" ]
echo 'PASS: native round-trip for three Agents and verified launcher/offline cleanup'
