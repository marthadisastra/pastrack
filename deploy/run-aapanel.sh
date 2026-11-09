#!/bin/bash
set -euo pipefail
cd /opt/pastrack
set -a
source /opt/pastrack/.env
set +a
exec /opt/pastrack/pastrack
