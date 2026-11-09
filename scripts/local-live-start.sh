#!/bin/sh
# Local UI and SQL access share the live application data and account privileges.
set -eu
TASK_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$TASK_ROOT"
TASK_CONFIG="$TASK_ROOT/.local/live-view.env"
[ -f "$TASK_CONFIG" ] || { echo 'Missing private .local/live-view.env; provision the read-only connection first.' >&2; exit 1; }
[ "$(stat -f '%Lp' "$TASK_CONFIG")" = 600 ] || { echo 'Private live config must have mode 600.' >&2; exit 1; }
mkdir -p .local/run .local/bin
chmod 700 .local .local/run
if lsof -nP -iTCP:18090 -sTCP:LISTEN >/dev/null 2>&1; then echo 'Port 18090 is in use; stop the existing local application first.' >&2; exit 1; fi
TASK_SSH_KEY=${PASTRACK_SSH_KEY:-/Users/ipidev/.ssh/imsa_agent_deploy_ed25519}
TASK_SOCKET="$TASK_ROOT/.local/run/live-db-tunnel.sock"
TASK_TUNNEL_PID=''
TASK_WEB_PID=''
cleanup(){
 if [ -n "$TASK_WEB_PID" ]; then kill -TERM "$TASK_WEB_PID" 2>/dev/null || true; fi
 if [ -n "$TASK_TUNNEL_PID" ]; then kill -TERM "$TASK_TUNNEL_PID" 2>/dev/null || true; fi
}
trap cleanup EXIT INT TERM
ssh -i "$TASK_SSH_KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -M -S "$TASK_SOCKET" -N -L 127.0.0.1:33308:127.0.0.1:3306 root@76.13.195.48 >.local/live-tunnel.log 2>&1 &
TASK_TUNNEL_PID=$!
echo "$TASK_TUNNEL_PID" >.local/run/live-tunnel.pid
TASK_TRIES=0
until ssh -S "$TASK_SOCKET" -O check root@76.13.195.48 >/dev/null 2>&1; do
 TASK_TRIES=$((TASK_TRIES+1)); [ "$TASK_TRIES" -lt 20 ] || { echo 'SSH tunnel unavailable; inspect .local/live-tunnel.log privately.' >&2; exit 1; }
 sleep 1
done
set -a
. "$TASK_CONFIG"
set +a
# Never run migrations, seeds, local bootstrap, or a second GPS receiver in this mode.
unset LOCAL_BOOTSTRAP_PASSWORD_FILE LOCAL_BOOTSTRAP_USER
go build -o .local/bin/pastrack-live-view ./cmd/pastrack-local
.local/bin/pastrack-live-view >.local/live-web.log 2>&1 &
TASK_WEB_PID=$!
echo "$TASK_WEB_PID" >.local/run/web.pid
TASK_TRIES=0
until curl -fsS http://127.0.0.1:18090/healthz >/dev/null 2>&1; do
 TASK_TRIES=$((TASK_TRIES+1)); [ "$TASK_TRIES" -lt 20 ] || { echo 'Local live-data UI unavailable; inspect .local/live-web.log privately.' >&2; exit 1; }
 sleep 1
done
echo 'Local live-data view: http://127.0.0.1:18090 — use the same account as the server; changes are live.'
wait "$TASK_WEB_PID"
