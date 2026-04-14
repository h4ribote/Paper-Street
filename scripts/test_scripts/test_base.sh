#!/bin/bash
# test_base.sh: Helper to load .env and run a python script
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd "$DIR/.."

if [ ! -f .env ]; then
  echo "Error: scripts/.env not found. Please run scripts/run_server_no_bots.sh first."
  exit 1
fi

# Load variables
export $(grep -v '^#' .env | xargs)

# Setup virtual environment if it doesn't exist
if [ ! -d "python_client/venv" ]; then
    echo "Setting up Python virtual environment..."
    python3 -m venv python_client/venv
    source python_client/venv/bin/activate
    pip install -r python_client/requirements.txt
else
    source python_client/venv/bin/activate
fi

# Run the passed python script
python "$@"
