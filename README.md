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
*   `ADMIN_PASSWORD` (必須): ボットが API キーを取得するための管理者用パスワード。
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

ボット用には `ADMIN_PASSWORD` を設定してください。全ボットは 1 つのコンテナ内で起動し、役割ごとの `BOT_ROLE` は内部で付与されます。
`API_KEY_FILE` を指定すると取得した API キーをファイルに保存します。

#### ボットの起動方式とリソース目安

* **個別サービス（従来の構成）**: コンテナごとにプロセスが分かれるため、停止・再起動やスケールが簡単ですが、各コンテナのランタイムやログ周りのオーバーヘッド分だけメモリが増えます。
* **1つのコンテナで全ボット（現在の構成）**: コンテナのオーバーヘッドが減るため **メモリ使用量は小さくなる傾向**があります。一方で、1プロセスの障害が全ボット停止につながりやすく、運用上の分離性は下がります。

結論として、**リソース最小化だけを重視するなら単一コンテナが有利**ですが、可用性・運用性を考えると個別サービスの方が扱いやすい構成です。

## ドキュメント
プロジェクトの詳細な仕様については [GAME_DESIGN.md](./docs/GAME_DESIGN.md) を参照してください。
