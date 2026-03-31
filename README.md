# Paper Street

**Real-time Financial MMO Simulation**

[詳細設計ドキュメント (GAME_DESIGN.md) はこちら](./docs/GAME_DESIGN.md)

[APIエンドポイント一覧 (API_ENDPOINTS.md) はこちら](./docs/API_ENDPOINTS.md)

[WebSocket仕様書 (WEBSOCKET.md) はこちら](./docs/WEBSOCKET.md)

[ドキュメント一覧 (docs/README.md) はこちら](./docs/README.md)

[ファイル構成 (FILE_STRUCTURE.md) はこちら](./docs/FILE_STRUCTURE.md)

## 概要 (Overview)
Paper Street は、「Wall Street Junior」などの金融シミュレーションゲームにインスパイアされた、Webブラウザベースの**リアルタイム金融MMO**です。
プレイヤーはプロフェッショナルな機関投資家として、高度な情報端末（The Terminal）を駆使し、ボットや他プレイヤーがひしめく市場で資産を競い合います。

## 特徴 (Key Features)

*   **Global Single Market (MMO)**:
    全プレイヤーが接続する単一の市場サーバー。あなたの注文が板（Order Book）に並び、市場価格を動かします。
*   **Advanced AI Ecosystem**:
    Market Maker, Trend Follower, HFT, Whale（大口）など、多様なアルゴリズムを持つボット群がリアルな流動性とボラティリティを生み出します。
*   **The Terminal UI**:
    ブルームバーグ端末のようなプロフェッショナルなUI。チャート、板情報、歩み値、ニュースフィードを自由にレイアウト可能。
*   **Seasonal Cycles**:
    2ヶ月ごとのシーズン制。シーズンごとに「大恐慌」や「バブル」などのテーマが変わり、ランキング上位者には永続的な称号が与えられます。

## 技術スタック (Tech Stack)

詳細は [GAME_DESIGN.md](./docs/GAME_DESIGN.md) 参照。

*   **Frontend**: Vanilla JS + Tailwind CSS / Lightweight Charts
*   **Backend**: Go (Goroutine + Channel ベースのインメモリ・マッチングエンジン)
*   **Database**: MySQL
*   **Infra**: Docker & Docker Compose (App + MySQL + Bot Containers)

## 実行方法 (Getting Started)

### 1. 事前準備

*   Docker / Docker Compose をインストールしてください。

### 2. 環境変数 (.env)

`deployments/.env` を作成して以下を設定します。

*   `MYSQL_ROOT_PASSWORD` (必須): MySQL の root パスワード。
*   `API_KEYS` (必須): 20 文字の 16 進数キーをカンマ区切りで指定します。
    *   例: `API_KEYS=0123456789abcdef0123,abcdef0123456789abcd`
*   `MYSQL_DATABASE` (任意): DB 名。未設定の場合は `paperstreet`。

`init.sql` は MySQL コンテナ初回起動時に自動適用されます。

### 3. 起動

```bash
cd deployments
docker compose up --build
```

ボットも起動する場合は以下を実行します。

```bash
cd deployments
docker compose --profile bots up --build
```

## ドキュメント
プロジェクトの詳細な仕様については [GAME_DESIGN.md](./docs/GAME_DESIGN.md) を参照してください。
