import requests
import websocket
import json
import threading
import time

class PaperStreetClient:
    def __init__(self, base_url, api_key=None):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()
        if self.api_key:
            self.session.headers.update({'X-API-Key': self.api_key})

        self.ws_url = self.base_url.replace("http://", "ws://").replace("https://", "wss://") + "/ws"
        if self.api_key:
            self.ws_url += f"?api_key={self.api_key}"

        self.ws = None
        self.ws_thread = None
        self.ws_messages = []
        self.ws_callbacks = {}
        self.is_connected = False

    def _get(self, endpoint, params=None):
        url = f"{self.base_url}{endpoint}"
        response = self.session.get(url, params=params)
        response.raise_for_status()
        return response.json()

    def _post(self, endpoint, data=None):
        url = f"{self.base_url}{endpoint}"
        response = self.session.post(url, json=data)
        response.raise_for_status()
        return response.json() if response.text else None

    def _delete(self, endpoint, params=None):
        url = f"{self.base_url}{endpoint}"
        response = self.session.delete(url, params=params)
        response.raise_for_status()
        return response.json() if response.text else None

    # --- 1. Authentication ---
    def check_health(self):
        return self._get("/health")

    def get_me(self):
        return self._get("/api/users/me")

    # --- 2. Market Data ---
    def get_assets(self):
        return self._get("/api/assets")

    def get_asset(self, asset_id):
        return self._get(f"/api/assets/{asset_id}")

    def get_orderbook(self, asset_id, depth=20):
        return self._get(f"/api/market/orderbook/{asset_id}", params={"depth": depth})

    def get_candles(self, asset_id, timeframe="1m", limit=100, start_time=None, end_time=None):
        params = {"timeframe": timeframe, "limit": limit}
        if start_time: params["start_time"] = start_time
        if end_time: params["end_time"] = end_time
        return self._get(f"/api/market/candles/{asset_id}", params=params)

    def get_trades(self, asset_id):
        return self._get(f"/api/market/trades/{asset_id}")

    def get_ticker(self):
        return self._get("/api/market/ticker")

    def get_news(self):
        return self._get("/api/news")

    def get_macro_indicators(self):
        return self._get("/api/macro/indicators")

    def get_fx_theoretical(self):
        return self._get("/api/fx/theoretical")

    # --- 3. Trading & Orders ---
    def create_order(self, asset_id, side, order_type, quantity, price=None, stop_price=None, leverage=1, user_id=None):
        data = {
            "asset_id": asset_id,
            "side": side,
            "type": order_type,
            "quantity": quantity,
            "leverage": leverage
        }
        if price is not None: data["price"] = price
        if stop_price is not None: data["stop_price"] = stop_price
        if user_id is not None: data["user_id"] = user_id
        return self._post("/api/orders", data)

    def cancel_order(self, order_id, asset_id):
        return self._delete(f"/api/orders/{order_id}", params={"asset_id": asset_id})

    def get_orders(self, status=None, asset_id=None, user_id=None, limit=None, offset=None):
        params = {}
        if status: params["status"] = status
        if asset_id: params["asset_id"] = asset_id
        if user_id: params["user_id"] = user_id
        if limit is not None: params["limit"] = limit
        if offset is not None: params["offset"] = offset
        return self._get("/api/orders", params=params)

    def get_order(self, order_id, asset_id):
        return self._get(f"/api/orders/{order_id}", params={"asset_id": asset_id})

    # --- 4. Portfolio & Wallet ---
    def get_balances(self):
        return self._get("/api/portfolio/balances")

    def get_portfolio_assets(self):
        return self._get("/api/portfolio/assets")

    def get_positions(self):
        return self._get("/api/portfolio/positions")

    def get_history(self):
        return self._get("/api/portfolio/history")

    def get_performance(self):
        return self._get("/api/portfolio/performance")

    # --- 5. Progression & Missions ---
    def get_rank(self, user_id=None):
        params = {"user_id": user_id} if user_id else {}
        return self._get("/api/user/rank", params=params)

    def get_daily_missions(self, user_id=None):
        params = {"user_id": user_id} if user_id else {}
        return self._get("/api/missions/daily", params=params)

    def get_user_missions(self):
        return self._get("/api/user/missions")

    def complete_mission(self, mission_id, user_id=None):
        data = {"user_id": user_id} if user_id else {}
        return self._post(f"/api/missions/{mission_id}/complete", data=data)

    # --- 6. Contracts ---
    def get_contracts(self, user_id=None):
        params = {"user_id": user_id} if user_id else {}
        return self._get("/api/contracts", params=params)

    def get_contract(self, contract_id, user_id=None):
        params = {"user_id": user_id} if user_id else {}
        return self._get(f"/api/contracts/{contract_id}", params=params)

    def deliver_contract(self, contract_id, quantity, user_id=None):
        data = {"quantity": quantity}
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/contracts/{contract_id}/deliver", data=data)

    def get_user_contracts(self):
        return self._get("/api/user/contracts")

    # --- 7. Liquidity Pools & FX ---
    def get_pools(self):
        return self._get("/api/pools")

    def get_pool(self, pool_id):
        return self._get(f"/api/pools/{pool_id}")

    def add_liquidity(self, pool_id, base_amount, quote_amount, lower_tick, upper_tick, user_id=None):
        data = {
            "base_amount": base_amount,
            "quote_amount": quote_amount,
            "lower_tick": lower_tick,
            "upper_tick": upper_tick
        }
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/pools/{pool_id}/positions", data=data)

    def get_pool_positions(self):
        return self._get("/api/pools/positions")

    def remove_liquidity(self, position_id):
        return self._delete(f"/api/pools/positions/{position_id}")

    def swap(self, pool_id, from_currency, to_currency, amount, user_id=None):
        data = {
            "from_currency": from_currency,
            "to_currency": to_currency,
            "amount": amount
        }
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/pools/{pool_id}/swap", data=data)

    # --- 8. Margin Pools ---
    def get_margin_pools(self):
        return self._get("/api/margin/pools")

    def get_margin_pool(self, pool_id):
        return self._get(f"/api/margin/pools/{pool_id}")

    def supply_margin(self, pool_id, cash_amount=None, asset_amount=None, user_id=None):
        data = {}
        if cash_amount is not None: data["cash_amount"] = cash_amount
        if asset_amount is not None: data["asset_amount"] = asset_amount
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/margin/pools/{pool_id}/supply", data=data)

    def withdraw_margin(self, pool_id, cash_amount=None, asset_amount=None, user_id=None):
        data = {}
        if cash_amount is not None: data["cash_amount"] = cash_amount
        if asset_amount is not None: data["asset_amount"] = asset_amount
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/margin/pools/{pool_id}/withdraw", data=data)

    def get_margin_positions(self, user_id=None):
        params = {"user_id": user_id} if user_id else {}
        return self._get("/api/margin/positions", params=params)

    def topup_margin(self, position_id, amount, user_id=None):
        data = {"amount": amount}
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/margin/positions/{position_id}/topup", data=data)

    def get_margin_liquidations(self, user_id=None):
        params = {"user_id": user_id} if user_id else {}
        return self._get("/api/margin/liquidations", params=params)

    # --- 9. World Meta & Events & Simulation ---
    def get_current_season(self):
        return self._get("/api/world/seasons/current")

    def get_regions(self):
        return self._get("/api/world/regions")

    def get_companies(self):
        return self._get("/api/world/companies")

    def get_events(self):
        return self._get("/api/world/events")

    def get_capital_structure(self, company_id):
        return self._get(f"/api/companies/{company_id}/capital-structure")

    def initiate_financing(self, company_id, target_amount=None, reason=None):
        data = {}
        if target_amount is not None: data["target_amount"] = target_amount
        if reason is not None: data["reason"] = reason
        return self._post(f"/api/companies/{company_id}/financing/initiate", data=data)

    def authorize_buyback(self, company_id, budget=None):
        data = {}
        if budget is not None: data["budget"] = budget
        return self._post(f"/api/companies/{company_id}/buyback/authorize", data=data)

    def get_production_status(self, company_id):
        return self._get(f"/api/companies/{company_id}/production-status")

    def get_supply_chain(self, company_id):
        return self._get(f"/api/companies/{company_id}/supply-chain")

    def get_financials(self, company_id, limit=None):
        params = {}
        if limit is not None: params["limit"] = limit
        return self._get(f"/api/companies/{company_id}/financials", params=params)

    def get_dividends(self, company_id, limit=None):
        params = {}
        if limit is not None: params["limit"] = limit
        return self._get(f"/api/companies/{company_id}/dividends", params=params)

    def simulate_company(self, company_id, quarters=1):
        return self._post(f"/api/companies/{company_id}/simulate", data={"quarters": quarters})

    # --- 10. Leaderboard ---
    def get_leaderboard(self, limit=20):
        return self._get("/api/leaderboard", params={"limit": limit})

    # --- 11. Indices ---
    def get_indices(self):
        return self._get("/api/indices/")

    def get_index(self, asset_id):
        return self._get(f"/api/indices/{asset_id}")

    def create_index(self, asset_id, quantity=1, user_id=None):
        data = {"quantity": quantity}
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/indices/{asset_id}/create", data=data)

    def redeem_index(self, asset_id, quantity=1, user_id=None):
        data = {"quantity": quantity}
        if user_id: data["user_id"] = user_id
        return self._post(f"/api/indices/{asset_id}/redeem", data=data)

    # --- 12. Bonds ---
    def get_bonds(self):
        return self._get("/api/bonds")

    def get_bond(self, bond_id):
        return self._get(f"/api/bonds/{bond_id}")

    # --- WebSocket Methods ---
    def _on_message(self, ws, message):
        data = json.loads(message)
        self.ws_messages.append(data)
        topic = data.get("topic")
        if topic and topic in self.ws_callbacks:
            self.ws_callbacks[topic](data)

    def _on_error(self, ws, error):
        print(f"WebSocket Error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        self.is_connected = False
        print(f"WebSocket Closed: {close_status_code} {close_msg}")

    def _on_open(self, ws):
        self.is_connected = True
        print("WebSocket Connected")

    def connect_ws(self):
        websocket.enableTrace(False)
        self.ws = websocket.WebSocketApp(
            self.ws_url,
            on_open=self._on_open,
            on_message=self._on_message,
            on_error=self._on_error,
            on_close=self._on_close
        )
        self.ws_thread = threading.Thread(target=self.ws.run_forever)
        self.ws_thread.daemon = True
        self.ws_thread.start()

        # Wait for connection
        timeout = 5
        start_time = time.time()
        while not self.is_connected and time.time() - start_time < timeout:
            time.time()
            time.sleep(0.1)

    def disconnect_ws(self):
        if self.ws:
            self.ws.close()
        if self.ws_thread:
            self.ws_thread.join(timeout=2)

    def subscribe(self, topics):
        if not self.is_connected:
            raise Exception("WebSocket not connected")
        if isinstance(topics, str):
            topics = [topics]
        msg = {
            "op": "subscribe",
            "args": topics
        }
        self.ws.send(json.dumps(msg))

    def unsubscribe(self, topics):
        if not self.is_connected:
            raise Exception("WebSocket not connected")
        if isinstance(topics, str):
            topics = [topics]
        msg = {
            "op": "unsubscribe",
            "args": topics
        }
        self.ws.send(json.dumps(msg))

    def on(self, topic, callback):
        self.ws_callbacks[topic] = callback

    def get_ws_messages(self, clear=False):
        msgs = list(self.ws_messages)
        if clear:
            self.ws_messages = []
        return msgs
