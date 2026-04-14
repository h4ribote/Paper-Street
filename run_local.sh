#!/bin/bash
set -e

# ==========================================
# Paper Street: Local Setup & Run Script
# ==========================================
# This script sets up the MySQL database, builds the Go server, and runs it locally.

# --- Configuration (can be overridden by environment variables) ---
MYSQL_USER=${MYSQL_USER:-"root"}
# If MYSQL_PASSWORD is not set, use empty string
if [ -z "${MYSQL_PASSWORD+x}" ]; then
  MYSQL_PASSWORD=""
fi
MYSQL_HOST=${MYSQL_HOST:-"127.0.0.1"}
MYSQL_PORT=${MYSQL_PORT:-"3306"}
MYSQL_DATABASE=${MYSQL_DATABASE:-"paperstreet"}

ADMIN_PASSWORD=${ADMIN_PASSWORD:-"admin123"}
PORT=${PORT:-"8000"}

# --- Dependency Checks ---
command -v mysql >/dev/null 2>&1 || { echo >&2 "Error: 'mysql' command is required but it's not installed. Aborting."; exit 1; }
command -v go >/dev/null 2>&1 || { echo >&2 "Error: 'go' command is required but it's not installed. Aborting."; exit 1; }

echo "=========================================="
echo " Starting Paper Street Local Setup..."
echo "=========================================="

# --- 1. Database Setup ---
echo "Checking/Creating MySQL Database '${MYSQL_DATABASE}'..."

# Build base mysql command with auth
if [ -z "$MYSQL_PASSWORD" ]; then
    MYSQL_CMD="mysql -u ${MYSQL_USER} -h ${MYSQL_HOST} -P ${MYSQL_PORT}"
else
    MYSQL_CMD="mysql -u ${MYSQL_USER} -p${MYSQL_PASSWORD} -h ${MYSQL_HOST} -P ${MYSQL_PORT}"
fi

# Create database if it doesn't exist
$MYSQL_CMD -e "CREATE DATABASE IF NOT EXISTS \`${MYSQL_DATABASE}\` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

echo "Applying init.sql..."
$MYSQL_CMD "${MYSQL_DATABASE}" < init.sql

echo "Applying seed.sql..."
$MYSQL_CMD "${MYSQL_DATABASE}" < seed.sql

echo "Database setup complete."
echo "------------------------------------------"

# --- 2. Build Go Server ---
echo "Building Go server..."
go build -o server_bin ./cmd/paper-street-server
echo "Build complete. Executable: server_bin"
echo "------------------------------------------"

# --- 3. Run Server ---
echo "Starting Paper Street Server on port ${PORT}..."

# Export required environment variables
export ADMIN_PASSWORD
export PORT

# Construct DATABASE_DSN for the Go application
# Format: user:password@tcp(host:port)/dbname?parseTime=true
if [ -z "$MYSQL_PASSWORD" ]; then
    export DATABASE_DSN="${MYSQL_USER}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}?parseTime=true"
else
    export DATABASE_DSN="${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}?parseTime=true"
fi

echo "Using DATABASE_DSN=${DATABASE_DSN}"
echo "------------------------------------------"

# --- 4. Calculate API Key ---
echo "Calculating API Key..."
MM_API_KEY=$(echo -n "market_maker" | openssl dgst -sha256 -hmac "$ADMIN_PASSWORD" | sed 's/^.* //')
MM_API_KEY=${MM_API_KEY:0:20}
echo "Market Maker API Key: ${MM_API_KEY}"

ARB_API_KEY=$(echo -n "arbitrageur" | openssl dgst -sha256 -hmac "$ADMIN_PASSWORD" | sed 's/^.* //')
ARB_API_KEY=${ARB_API_KEY:0:20}
echo "Arbitrageur API Key: ${ARB_API_KEY}"

echo "------------------------------------------"

# Start the server
./server_bin
