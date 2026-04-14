# AI Agent Development & Testing Guide

This document provides guidelines and standard operating procedures for AI coding agents working on the Paper Street repository.

## 1. Local Testing Environment

When modifying server logic, bug fixing, or adding new features, you **must** verify your changes using the provided testing environment before concluding your task.

### Starting the Server (Clean Environment)

We have a dedicated script to start the server without any background bots (Market Maker, etc.), giving you a clean environment to test specific logic and API responses.

1. Start the server using the setup script:
   ```bash
   ./scripts/run_server_no_bots.sh
   ```
   *Note: You should run this in a background process or a separate terminal session if you are executing other bash commands.*

2. What the script does:
   - Resets and initializes the MySQL database (`init.sql`, `seed.sql`).
   - Calculates authentication API Keys for testing.
   - Saves these keys and the base URL into `scripts/.env`.
   - Builds and starts the Go server on port 8000.

### Using the Python Test Scripts

Once the server is running, you can use the wrapper shell scripts located in `scripts/test_scripts/` to interact with the API. These scripts automatically handle python virtual environment setup and read from `scripts/.env`.

**Available Scripts & Usage:**

*   **`test_market_data.sh`**:
    *   Usage: `./scripts/test_scripts/test_market_data.sh [health|assets|asset|orderbook|ticker|news] [--asset_id ID] [--depth DEPTH]`
    *   Example: `./scripts/test_scripts/test_market_data.sh orderbook --asset_id 1`

*   **`test_orders.sh`**:
    *   Usage: `./scripts/test_scripts/test_orders.sh [list|create|cancel|get] [--side BUY/SELL] [--type MARKET/LIMIT/STOP/STOP_LIMIT] [--price PRICE] [--quantity QTY] ...`
    *   Example: `./scripts/test_scripts/test_orders.sh create --asset_id 1 --side BUY --type LIMIT --price 100 --quantity 10`

*   **`test_portfolio.sh`**:
    *   Usage: `./scripts/test_scripts/test_portfolio.sh [balances|assets|positions|history|performance]`
    *   Example: `./scripts/test_scripts/test_portfolio.sh balances`

*   **`test_pools.sh`**:
    *   Usage: `./scripts/test_scripts/test_pools.sh [list|get|margin_list|margin_get] [--pool_id ID]`
    *   Example: `./scripts/test_scripts/test_pools.sh list`

*   **`test_world.sh`**:
    *   Usage: `./scripts/test_scripts/test_world.sh [season|regions|companies|events|leaderboard]`
    *   Example: `./scripts/test_scripts/test_world.sh season`

*   **`test_websocket.sh`**:
    *   Usage: `./scripts/test_scripts/test_websocket.sh [topic] [--duration SECONDS]`
    *   Example: `./scripts/test_scripts/test_websocket.sh market.ticker --duration 5`

## 2. Core Directives for Agents

1. **Test-Driven Verification:** After making any backend code changes, start the server using `./scripts/run_server_no_bots.sh &` and use the appropriate `test_*.sh` script to confirm your change works as expected.
2. **Clean up:** If you start the server in the background, remember to kill the process (`kill $(lsof -t -i :8000)`) when you are done or before restarting it.
3. **Database State:** Keep in mind that restarting `./scripts/run_server_no_bots.sh` completely resets the database.
4. **Python Client Extension:** If you add new API endpoints to the Go server, make sure to update `scripts/python_client/client.py` and the relevant runner scripts so they can be easily tested.
