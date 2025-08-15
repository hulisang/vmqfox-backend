#!/bin/sh
set -eu

# Working directory should be app root
cd /var/www/html

ENV_FILE=".env"

# Generate .env file in the same format as env.example
cat > "$ENV_FILE" <<EOF
APP_DEBUG = ${APP_DEBUG:-true}
APP_TRACE = ${APP_TRACE:-false}
APP_FRONTEND_URL = ${APP_FRONTEND_URL:-http://localhost:3006}

[DATABASE]
TYPE = ${DB_TYPE:-mysql}
HOSTNAME = ${DB_HOSTNAME:-localhost}
DATABASE = ${DB_DATABASE:-vmq}
USERNAME = ${DB_USERNAME:-root}
PASSWORD = ${DB_PASSWORD:-}
HOSTPORT = ${DB_HOSTPORT:-3306}
CHARSET = ${DB_CHARSET:-utf8}
PREFIX = ${DB_PREFIX:-}
DEBUG = ${DB_DEBUG:-true}

[REDIS]
HOST = ${REDIS_HOST:-127.0.0.1}
PORT = ${REDIS_PORT:-6379}
PASSWORD = ${REDIS_PASSWORD:-}
SELECT = ${REDIS_SELECT:-0}

[CACHE]
DRIVER = ${CACHE_DRIVER:-file}

[SESSION]
DRIVER = ${SESSION_DRIVER:-file}
EOF

echo "=== Generated .env file content ==="
cat "$ENV_FILE"
echo "=== End of .env file ==="

# Ensure runtime dir exists (ThinkPHP) and is writable
mkdir -p runtime

echo "Starting vmqfox-backend with command: $@"

# Execute passed command (default defined in CMD)
exec "$@"

