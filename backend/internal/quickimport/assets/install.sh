#!/bin/sh
set -eu
action=${1:-}
agent=${2:-}
server=${3:-}
ticket=${4:-}
case "$agent" in claude|codex|opencode) ;; *) echo 'Unsupported Agent' >&2; exit 1;; esac
if ! command -v python3 >/dev/null 2>&1 || ! python3 -c 'import sys; sys.exit(0 if sys.version_info >= (3,11) else 1)'; then
  echo 'Install Python 3.11 or later, then retry.' >&2; exit 1
fi
case "$action" in
  clean)
    runner="$HOME/.sub2api-quick-import/$agent/restore.py"
    test -f "$runner" || { echo 'No local recovery script for this Agent.' >&2; exit 1; }
    python3 "$runner" clean --agent "$agent"
    ;;
  install)
    case "$server" in https://*) ;; *) echo 'HTTPS server required' >&2; exit 1;; esac
    runner=$(mktemp)
    trap 'rm -f "$runner"' EXIT HUP INT TERM
    curl --fail --silent --show-error --proto '=https' --max-time 30 "${server%/}/api/v1/quick-import/assets/installer.py" -o "$runner"
    python3 "$runner" install --agent "$agent" --server "$server" --ticket "$ticket"
    ;;
  *) echo 'Expected install or clean' >&2; exit 1;;
esac
