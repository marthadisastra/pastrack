#!/bin/sh
# Runs only isolated loopback services. Never reads production deployment files.
set -eu
TASK_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$TASK_ROOT"
MYSQL_BASE=${MYSQL_BASE:-/Applications/AMPPS/apps/mysql}
mkdir -p .local/mysql .local/run .local/bin .local/spool
chmod 700 .local .local/mysql .local/run .local/spool
if [ ! -f .local/mysql/auto.cnf ]; then
 "$MYSQL_BASE/bin/mysqld" --no-defaults --initialize-insecure --basedir="$MYSQL_BASE" --datadir="$TASK_ROOT/.local/mysql" --log-error="$TASK_ROOT/.local/mysql-init.log"
fi
if ! "$MYSQL_BASE/bin/mysqladmin" --protocol=TCP -h 127.0.0.1 -P 33307 -u root ping >/dev/null 2>&1; then
 nohup "$MYSQL_BASE/bin/mysqld" --no-defaults --basedir="$MYSQL_BASE" --datadir="$TASK_ROOT/.local/mysql" --bind-address=127.0.0.1 --port=33307 --socket="$TASK_ROOT/.local/run/mysql.sock" --pid-file="$TASK_ROOT/.local/run/mysql.pid" --log-error="$TASK_ROOT/.local/mysql.log" --mysqlx=OFF >.local/mysql-console.log 2>&1 &
 TASK_ATTEMPTS=0
 until "$MYSQL_BASE/bin/mysqladmin" --protocol=TCP -h 127.0.0.1 -P 33307 -u root ping >/dev/null 2>&1; do
  TASK_ATTEMPTS=$((TASK_ATTEMPTS+1)); [ "$TASK_ATTEMPTS" -lt 30 ] || exit 1
  sleep 1
 done
fi
# Verify this is our private instance, rather than reusing an unrelated listener.
TASK_DATADIR=$("$MYSQL_BASE/bin/mysql" --protocol=TCP -h 127.0.0.1 -P 33307 -u root -N -e 'SELECT @@datadir')
[ "$TASK_DATADIR" = "$TASK_ROOT/.local/mysql/" ] || { echo 'Unexpected database instance; refusing startup.' >&2; exit 1; }
export DB_HOST=127.0.0.1 DB_PORT=33307 DB_USER=root DB_NAME=pastrack_dev_local
export WEB_ADDR=127.0.0.1:18090 GPS_HOST=127.0.0.1 GPS_PORT=15001 HEALTH_ADDR=127.0.0.1:18091
export PASTRACK_SCHEMA=v2 SPOOL_DIR="$TASK_ROOT/.local/spool" REALTIME_SOCKET_PATH="$TASK_ROOT/.local/run/live.sock"
export LOCAL_BOOTSTRAP_USER=admin LOCAL_BOOTSTRAP_PASSWORD_FILE="$TASK_ROOT/.local/bootstrap-password"
python3 - <<'PY'
import secrets,os
from pathlib import Path
p=Path('.local/bootstrap-password')
if not p.exists():
 p.write_text(secrets.token_urlsafe(24)+'\n');p.chmod(0o600)
PY
go run ./cmd/pastrack-db
go build -o .local/bin/pastrack-local ./cmd/pastrack-local
go build -o .local/bin/pastrack-receiver .
for TASK_SERVICE in web receiver; do
 if [ -f ".local/run/$TASK_SERVICE.pid" ] && kill -0 "$(cat ".local/run/$TASK_SERVICE.pid")" 2>/dev/null; then
  echo "$TASK_SERVICE already running; use local-stop.sh before restarting rebuilt binaries." >&2
  exit 1
 fi
done
if [ -S "$REALTIME_SOCKET_PATH" ]; then
 if lsof -t "$REALTIME_SOCKET_PATH" >/dev/null 2>&1; then
  echo 'Existing realtime socket is owned by another process; refusing startup.' >&2
  exit 1
 fi
 rm "$REALTIME_SOCKET_PATH"
fi
nohup .local/bin/pastrack-local >.local/web.log 2>&1 &
echo $! >.local/run/web.pid
TASK_ATTEMPTS=0
until curl -fsS http://127.0.0.1:18090/healthz >/dev/null; do
 TASK_ATTEMPTS=$((TASK_ATTEMPTS+1)); [ "$TASK_ATTEMPTS" -lt 20 ] || exit 1
 sleep 1
done
nohup .local/bin/pastrack-receiver >.local/receiver.log 2>&1 &
echo $! >.local/run/receiver.pid
echo 'PasTrack: http://127.0.0.1:18090 — login admin; private password file: .local/bootstrap-password'

# Foreground supervisor keeps both development services alive.
trap './scripts/local-stop.sh' INT TERM
wait
