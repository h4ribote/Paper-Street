# AI Agent Development & Testing Guide

This document provides guidelines and standard operating procedures for AI coding agents working on the Paper Street repository.

## 1. Local Testing Environment

When modifying server logic, bug fixing, or adding new features, you **must** verify your changes using the provided testing environment before concluding your task.

### Starting the Server (Clean Environment)

We have a dedicated script to start the server without any background bots (Market Maker, etc.), giving you a clean environment to test specific logic and API responses.

1. Start the server using the setup script:
   ```bash
   ./dev/run_server_no_bots.sh
   ```
   *Note: You should run this in a background process or a separate terminal session if you are executing other bash commands.*

2. What the script does:
   - Resets and initializes the MySQL database (`init.sql`, `seed.sql`).
   - Calculates authentication API Keys for testing.
   - Saves these keys and the base URL into `dev/.env`.
   - Builds and starts the Go server on port 8000.

### Using the Python Test Scripts

Once the server is running, you can use the wrapper shell scripts located in `dev/test_scripts/` to interact with the API. These scripts automatically handle python virtual environment setup and read from `dev/.env`.

**Available Scripts & Usage:**

*   **`test_market_data.sh`**:
    *   Usage: `./dev/test_scripts/test_market_data.sh [health|assets|asset|orderbook|ticker|news] [--asset_id ID] [--depth DEPTH]`
    *   Example: `./dev/test_scripts/test_market_data.sh orderbook --asset_id 1`

*   **`test_orders.sh`**:
    *   Usage: `./dev/test_scripts/test_orders.sh [list|create|cancel|get] [--side BUY/SELL] [--type MARKET/LIMIT/STOP/STOP_LIMIT] [--price PRICE] [--quantity QTY] ...`
    *   Example: `./dev/test_scripts/test_orders.sh create --asset_id 1 --side BUY --type LIMIT --price 100 --quantity 10`

*   **`test_portfolio.sh`**:
    *   Usage: `./dev/test_scripts/test_portfolio.sh [balances|assets|positions|history|performance]`
    *   Example: `./dev/test_scripts/test_portfolio.sh balances`

*   **`test_pools.sh`**:
    *   Usage: `./dev/test_scripts/test_pools.sh [list|get|margin_list|margin_get] [--pool_id ID]`
    *   Example: `./dev/test_scripts/test_pools.sh list`

*   **`test_world.sh`**:
    *   Usage: `./dev/test_scripts/test_world.sh [season|regions|companies|events|leaderboard]`
    *   Example: `./dev/test_scripts/test_world.sh season`

*   **`test_websocket.sh`**:
    *   Usage: `./dev/test_scripts/test_websocket.sh [topic] [--duration SECONDS]`
    *   Example: `./dev/test_scripts/test_websocket.sh market.ticker --duration 5`

## 2. Core Directives for Agents

1. **Test-Driven Verification:** After making any backend code changes, start the server using `./dev/run_server_no_bots.sh &` and use the appropriate `test_*.sh` script to confirm your change works as expected.
2. **Clean up:** If you start the server in the background, remember to kill the process (`kill $(lsof -t -i :8000)`) when you are done or before restarting it.
3. **Database State:** Keep in mind that restarting `./dev/run_server_no_bots.sh` completely resets the database.
4. **Python Client Extension:** If you add new API endpoints to the Go server, make sure to update `dev/python_client/client.py` and the relevant runner scripts so they can be easily tested.
5. **Create Custom Test Scripts:** If necessary, add test scripts reflecting specific operational procedures by editing the files in `python_client/`, `test_scripts/`, and updating `Guide.md` accordingly.
6. **Autonomous Execution:** Proceed with tasks directly without asking the user for plan approval.
7. **Comprehensive Modifications:** When modifying code, check if there are other places using the same syntax or logic as the target of your modification, and determine whether additional modifications are necessary across the codebase.
8. **Update and Add Unit Tests:** When modifying or adding backend logic, update or create corresponding `*_test.go` files to maintain code quality and prevent regressions.
9. **Proper Error Handling and Logging:** Implement thorough error handling and include sufficient logging (using the internal `logger` package) to facilitate debugging and system monitoring.
10. **Sync with Documentation:** Ensure that any changes to API endpoints, database schemas, or core system logic are reflected in the corresponding documentation within the `docs/` directory.
