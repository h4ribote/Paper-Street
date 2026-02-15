# Paper Street

**Real-time Financial MMO Simulation**

[詳細設計ドキュメント (GAME_DESIGN.md) はこちら](./docs/GAME_DESIGN.md)

[APIエンドポイント一覧 (API_ENDPOINTS.md) はこちら](./docs/API_ENDPOINTS.md)

[WebSocket仕様書 (WEBSOCKET.md) はこちら](./docs/WEBSOCKET.md)

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

現在設計段階ですが、以下の構成を予定しています（詳細は `GAME_DESIGN.md` 参照）。

*   **Frontend**: Vanilla JS + Tailwind CSS / Lightweight Charts
*   **Backend**: Python (FastAPI)
*   **Database**: MySQL
*   **Infra**: Docker & Docker Compose (Microservices for Bots)

## ドキュメント
プロジェクトの詳細な仕様については [GAME_DESIGN.md](./docs/GAME_DESIGN.md) を参照してください。
