# API Endpoints

Paper Street のバックエンドAPIエンドポイント一覧です。
詳細は各設計ドキュメントを参照してください。

## 1. Authentication (認証)
*   `GET /auth/login`
    *   Discord OAuth2 ログインプロセスを開始します。
*   `GET /auth/callback`
    *   Discord からのコールバックを受け取り、JWTトークンを発行します。
*   `POST /auth/refresh`
    *   リフレッシュトークンを使用して新しいアクセストークンを取得します。
*   `POST /auth/logout`
    *   ログアウトし、トークンを無効化します。
*   `GET /users/me`
    *   現在のユーザー情報を取得します。

## 2. Market Data (市場データ)
*   `GET /assets`
    *   全銘柄リストを取得します。フィルタリング（セクター、タイプ等）が可能です。
*   `GET /assets/{asset_id}`
    *   指定した銘柄の詳細情報を取得します。
*   `GET /market/orderbook/{asset_id}`
    *   指定した銘柄の板情報（Order Book）を取得します。
*   `GET /market/candles/{asset_id}`
    *   指定した銘柄のローソク足データを取得します。パラメータ: `timeframe`, `limit`, `start_time`, `end_time`。
*   `GET /market/trades/{asset_id}`
    *   指定した銘柄の歩み値（約定履歴）を取得します。
*   `GET /market/ticker`
    *   全銘柄の現在値と変動率などの概要を取得します。
*   `GET /news`
    *   ニュースフィードを取得します。
*   `GET /macro/indicators`
    *   各国のマクロ経済指標を取得します。

## 3. Trading & Orders (取引・注文)
*   `POST /orders`
    *   新規注文を発注します。
    *   Body: `asset_id`, `side` (BUY/SELL), `type` (MARKET/LIMIT/STOP), `quantity`, `price` (LIMITの場合)。
*   `DELETE /orders/{order_id}`
    *   指定した注文をキャンセルします。
*   `GET /orders`
    *   注文一覧を取得します。ステータス（OPEN/FILLED/CANCELLED）でフィルタリング可能です。
*   `GET /orders/{order_id}`
    *   注文の詳細情報を取得します。

## 4. Portfolio & Wallet (ポートフォリオ・資産)
*   `GET /portfolio/balances`
    *   通貨残高（Cash）を取得します。
*   `GET /portfolio/assets`
    *   保有資産（現物）の一覧を取得します。
*   `GET /portfolio/positions`
    *   現在の建玉（信用ポジション）一覧を取得します。
*   `GET /portfolio/history`
    *   取引履歴、入出金履歴、配当などのログを取得します。
*   `GET /portfolio/performance`
    *   資産推移グラフ用データを取得します。

## 5. Liquidity Pools & FX (流動性プール・FX)
*   `GET /pools`
    *   流動性プールの一覧を取得します。
*   `GET /pools/{pool_id}`
    *   指定したプールの詳細情報（流動性、手数料、現在のTickなど）を取得します。
*   `POST /pools/{pool_id}/positions`
    *   流動性を提供し、ポジションを作成します（Concentrated Liquidity）。
*   `GET /pools/positions`
    *   ユーザーの流動性ポジション一覧を取得します。
*   `DELETE /pools/positions/{position_id}`
    *   流動性を解除し、手数料と元本を回収します。
*   `POST /pools/{pool_id}/swap`
    *   プールを介して通貨のスワップを行います。

## 6. Margin Pools (信用取引プール)
*   `GET /margin/pools`
    *   信用取引（貸株・融資）プールの一覧を取得します。
*   `GET /margin/pools/{pool_id}`
    *   プールの詳細（金利、在庫状況）を取得します。
*   `POST /margin/pools/{pool_id}/supply`
    *   資金または株式を供給し、金利収入を得ます。
*   `POST /margin/pools/{pool_id}/withdraw`
    *   供給した資金または株式を引き出します。

## 7. World Meta & Events (ゲーム世界情報)
*   `GET /world/seasons/current`
    *   現在のシーズン情報（テーマ、終了日時など）を取得します。
*   `GET /world/regions`
    *   地域と国家のリストを取得します。
*   `GET /world/companies`
    *   企業リストと詳細情報を取得します。
*   `GET /world/events`
    *   予定されているイベントや過去のイベントログを取得します。

## 8. Leaderboard (ランキング)
*   `GET /leaderboard`
    *   資産ランキングを取得します。シーズン別、通算などのフィルタが可能です。
