# API Endpoints

Paper Street のバックエンドAPIエンドポイント一覧です。
詳細は各設計ドキュメントを参照してください。

## 1. Authentication (認証)
*   **APIキー認証**:
    *   APIキーは10バイトのバイナリを20文字の16進数で表現します。
    *   HTTPヘッダー `X-API-Key` に20文字の16進数キーを指定します。
*   `GET /health`
    *   稼働確認用のヘルスチェックです（認証不要）。
*   `POST /auth/login` / `POST /auth/bot`
    *   管理者用パスワードとボットの役割からAPIキーを取得します。
    *   Body: `{ "role": "market_maker", "admin_password": "..." }`
*   `GET /users/me`
    *   現在のユーザー情報を取得します。`user_id` を指定した場合はそのユーザーを返します。

## 2. Market Data (市場データ)
*   `GET /assets`
    *   全銘柄リストを取得します。フィルタリング（セクター、タイプ等）が可能です。
*   `GET /assets/{asset_id}`
    *   指定した銘柄の詳細情報を取得します。
*   `GET /market/orderbook/{asset_id}`
    *   指定した銘柄の板情報（Order Book）を取得します。`depth` で板の段数を指定できます（デフォルト20、最大100）。
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
*   `GET /fx/theoretical`
    *   マクロ指標に基づく理論FXレート（Local/ARC）を取得します。

## 3. Trading & Orders (取引・注文)
*   `POST /orders`
    *   新規注文を発注します。
    *   Body: `asset_id`, `side` (BUY/SELL), `type` (MARKET/LIMIT/STOP/STOP_LIMIT), `quantity`。
    *   `price` は LIMIT/STOP_LIMIT の場合に必須、`stop_price` は STOP/STOP_LIMIT の場合に必須です。
    *   `user_id` は任意（APIキーに紐づくユーザーがある場合は省略可能）です。
    *   `leverage` は任意（デフォルト1）。2以上を指定すると分離マージンの証拠金でポジションが作成されます。
*   `DELETE /orders/{order_id}`
    *   指定した注文をキャンセルします。`asset_id` クエリパラメータが必須です。
*   `GET /orders`
    *   注文一覧を取得します。ステータス（OPEN/PARTIAL/FILLED/CANCELLED/REJECTED）でフィルタリング可能です。
    *   `asset_id` と `user_id` でも絞り込みできます。`limit` と `offset` でページネーションできます。
*   `GET /orders/{order_id}`
    *   注文の詳細情報を取得します。`asset_id` クエリパラメータが必須です。

## 4. Portfolio & Wallet (ポートフォリオ・資産)
*   `GET /portfolio/balances`
    *   通貨残高（Cash）を取得します。
*   `GET /portfolio/assets`
    *   保有資産（現物）の一覧を取得します。
*   `GET /portfolio/positions`
    *   現在の建玉（信用ポジション）一覧を取得します。
*   `GET /portfolio/history`
    *   取引の約定履歴を取得します。
*   `GET /portfolio/performance`
    *   現在時点の資産評価スナップショットを取得します。

## 5. Progression & Missions (進行・ミッション)
*   `GET /user/rank`
    *   現在のランク/XP情報を取得します。`user_id` を指定しない場合は認証ユーザーを参照します。
*   `GET /missions/daily`
    *   本日のデイリーミッション一覧と達成状況を取得します。`user_id` を指定しない場合は認証ユーザーを参照します。
*   `GET /user/missions`
    *   `GET /missions/daily` と同様に、当日のミッション進捗を返します。
*   `POST /missions/{mission_id}/complete`
    *   指定ミッションの達成を報告します（フロント側で達成検知後に送信）。Body: `user_id` (任意)。

## 6. Contracts (大口コントラクト)
*   `GET /contracts`
    *   募集中のコントラクト一覧を取得します。`user_id` を指定するとユーザーの納品状況を含みます。
*   `GET /contracts/{contract_id}`
    *   指定コントラクトの詳細を取得します。`user_id` を指定するとユーザーの納品状況を含みます。
*   `POST /contracts/{contract_id}/deliver`
    *   コントラクトへ納品します。Body: `quantity`, `user_id` (任意)。
*   `GET /user/contracts`
    *   `GET /contracts` と同様に、ユーザーの納品状況を返します。

## 7. Liquidity Pools & FX (流動性プール・FX)
*   `GET /pools`
    *   流動性プールの一覧を取得します。
*   `GET /pools/{pool_id}`
    *   指定したプールの詳細情報（流動性、手数料、現在のTickなど）を取得します。
*   `POST /pools/{pool_id}/positions`
    *   流動性を提供し、ポジションを作成します（Concentrated Liquidity）。
    *   Body: `base_amount`, `quote_amount`, `lower_tick`, `upper_tick`, `user_id` (任意)。
*   `GET /pools/positions`
    *   ユーザーの流動性ポジション一覧を取得します。
*   `DELETE /pools/positions/{position_id}`
    *   流動性を解除し、手数料と元本を回収します。
*   `POST /pools/{pool_id}/swap`
    *   プールを介して通貨のスワップを行います。
    *   Body: `from_currency`, `to_currency`, `amount`, `user_id` (任意)。
    *   `pool_id` に `0` を指定した場合、Routerが最適なルート（Direct / ARCマルチホップ + Fee Tier 分割）を自動選択します。

## 8. Margin Pools (信用取引プール)
*   `GET /margin/pools`
    *   信用取引（貸株・融資）プールの一覧を取得します。
*   `GET /margin/pools/{pool_id}`
    *   プールの詳細（金利、在庫状況）を取得します。
*   `POST /margin/pools/{pool_id}/supply`
    *   資金または株式を供給し、金利収入を得ます。
    *   Body: `cash_amount`, `asset_amount`, `user_id` (任意)。
*   `POST /margin/pools/{pool_id}/withdraw`
    *   供給した資金または株式を引き出します。
    *   Body: `cash_amount`, `asset_amount`, `user_id` (任意)。
*   `GET /margin/positions`
    *   分離マージンポジションの一覧を取得します。`user_id` を指定するとユーザーに限定します。
*   `POST /margin/positions/{position_id}/topup`
    *   既存のポジションに追証を行います。Body: `amount`, `user_id` (任意)。
*   `GET /margin/liquidations`
    *   強制決済の履歴を取得します。`user_id` を指定するとユーザーに限定します。

## 9. World Meta & Events (ゲーム世界情報)
*   `GET /world/seasons/current`
    *   現在のシーズン情報（テーマ、終了日時など）を取得します。
*   `GET /world/regions`
    *   地域と国家のリストを取得します。
*   `GET /world/companies`
    *   企業リストと詳細情報を取得します。
*   `GET /world/events`
    *   予定されているイベントや過去のイベントログを取得します。

## 9.1 Corporate Finance & Simulation (企業ファイナンス/シミュレーション)
*   `GET /companies/{company_id}/capital-structure`
    *   企業の発行済株式数・流通株式数・自己株式数と時価総額を取得します。
*   `POST /companies/{company_id}/financing/initiate`
    *   資金調達を開始します。Body: `target_amount` (任意), `reason` (任意)。
*   `POST /companies/{company_id}/buyback/authorize`
    *   自社株買いを実行します。Body: `budget` (任意)。
*   `GET /companies/{company_id}/production-status`
    *   生産能力・在庫・稼働率などの生産状況を取得します。
*   `GET /companies/{company_id}/supply-chain`
    *   生産レシピと原材料の構成を取得します。
*   `GET /companies/{company_id}/financials`
    *   企業の四半期決算データを取得します。`limit` で件数を指定できます。
*   `POST /companies/{company_id}/simulate`
    *   企業の四半期シミュレーションを実行します。Body: `quarters` (任意、デフォルト1)。

## 10. Leaderboard (ランキング)
*   `GET /leaderboard`
    *   資産ランキングを取得します。`limit` で件数を指定できます（デフォルト20）。

## 11. Indices (指数)
*   `POST /indices/{asset_id}/create`
    *   Index（指数）の構成銘柄（現物バスケット）を拠出し、Indexユニットを発行（Creation）します。
    *   **すべてのプレイヤーおよびBotが利用可能です。**
    *   Body: `quantity` (作成するIndexの単位数。デフォルトは1)、`user_id` (任意)。
*   `POST /indices/{asset_id}/redeem`
    *   保有しているIndexユニットを返還（償還）し、構成銘柄（現物バスケット）を受け取ります（Redemption）。
    *   **すべてのプレイヤーおよびBotが利用可能です。**
    *   Body: `quantity` (償還するIndexの単位数。デフォルトは1)、`user_id` (任意)。
