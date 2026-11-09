#!/bin/bash
set -euo pipefail
cd /www/wwwroot/pastrack
set -a
source /etc/pastrack/web.env
set +a
exec /www/wwwroot/pastrack/pastrack-web
