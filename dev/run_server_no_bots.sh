#!/bin/bash
set -e

# ==========================================
# Paper Street: Local Setup & Run Script (No Bots)
# ==========================================
# This script sets up the MySQL database, builds the Go server, and runs it locally without bots.
# It also generates API keys and saves them to an environment file for tests.

cd "$(dirname "$0")/.."

# --- Configuration ---
MYSQL_USER=${MYSQL_USER:-"root"}
if [ -z "${MYSQL_PASSWORD+x}" ]; then
  MYSQL_PASSWORD=""
fi
MYSQL_HOST=${MYSQL_HOST:-"127.0.0.1"}
MYSQL_PORT=${MYSQL_PORT:-"3306"}
MYSQL_DATABASE=${MYSQL_DATABASE:-"paperstreet"}
ADMIN_PASSWORD=${ADMIN_PASSWORD:-"admin123"}
PORT=${PORT:-"8000"}

# Check dependencies
command -v mysql >/dev/null 2>&1 || { echo >&2 "Error: 'mysql' command is required but it's not installed. Aborting."; exit 1; }
command -v go >/dev/null 2>&1 || { echo >&2 "Error: 'go' command is required but it's not installed. Aborting."; exit 1; }

echo "=========================================="
echo " Starting Paper Street Server (No Bots)   "
echo "=========================================="

# --- 1. Database Setup ---
echo "Checking/Creating MySQL Database '${MYSQL_DATABASE}'..."
if [ -z "$MYSQL_PASSWORD" ]; then
    MYSQL_CMD="mysql -u ${MYSQL_USER} -h ${MYSQL_HOST} -P ${MYSQL_PORT}"
else
    MYSQL_CMD="mysql -u ${MYSQL_USER} -p${MYSQL_PASSWORD} -h ${MYSQL_HOST} -P ${MYSQL_PORT}"
fi

$MYSQL_CMD -e "CREATE DATABASE IF NOT EXISTS \`${MYSQL_DATABASE}\` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

echo "Applying init.sql..."
$MYSQL_CMD "${MYSQL_DATABASE}" < init.sql

echo "Applying seed.sql..."
$MYSQL_CMD "${MYSQL_DATABASE}" < seed.sql
echo "Database setup complete."
echo "------------------------------------------"

# --- 2. Calculate API Keys ---
echo "Calculating API Keys for testing..."
MM_API_KEY=$(echo -n "market_maker" | openssl dgst -sha256 -hmac "$ADMIN_PASSWORD" | sed 's/^.* //')
MM_API_KEY=${MM_API_KEY:0:20}

ARB_API_KEY=$(echo -n "arbitrageur" | openssl dgst -sha256 -hmac "$ADMIN_PASSWORD" | sed 's/^.* //')
ARB_API_KEY=${ARB_API_KEY:0:20}

echo "Market Maker API Key: ${MM_API_KEY}"
echo "Arbitrageur API Key: ${ARB_API_KEY}"

# Write keys to dev/.env for python scripts
mkdir -p dev
cat <<EOF > dev/.env
PAPERSTREET_BASE_URL=http://localhost:${PORT}
PAPERSTREET_API_KEY=${MM_API_KEY}
PAPERSTREET_ARB_API_KEY=${ARB_API_KEY}
EOF

echo "Saved test environment variables to dev/.env"
echo "------------------------------------------"

# --- 3. Build Go Server ---
echo "Building Go server..."
go build -o server_bin ./cmd/paper-street-server
echo "Build complete."
echo "------------------------------------------"

# --- 4. Run Server ---
export ADMIN_PASSWORD
export PORT
if [ -z "$MYSQL_PASSWORD" ]; then
    export DATABASE_DSN="${MYSQL_USER}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}?parseTime=true"
else
    export DATABASE_DSN="${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}?parseTime=true"
fi

echo "Starting Server on port ${PORT}..."
echo "Press Ctrl+C to stop."
./server_bin
