#!/bin/sh
set -eu
TASK_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$TASK_ROOT"
for TASK_SERVICE in receiver web; do
 TASK_PID_FILE=".local/run/$TASK_SERVICE.pid"
 if [ -f "$TASK_PID_FILE" ]; then
  TASK_PID=$(cat "$TASK_PID_FILE")
  # Avoid signaling a reused PID for an unrelated process.
  TASK_COMMAND=$(ps -p "$TASK_PID" -o command= || true)
  case "$TASK_COMMAND" in *".local/bin/pastrack-"*) kill -TERM "$TASK_PID" 2>/dev/null || true;; esac
 fi
done
# MySQL remains available for inspection; no database or spool files are removed.
