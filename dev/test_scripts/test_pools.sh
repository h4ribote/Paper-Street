#!/bin/bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
"$DIR/test_base.sh" "$DIR/../python_client/run_pools.py" "$@"
