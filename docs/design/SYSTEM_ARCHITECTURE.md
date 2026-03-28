# System Architecture & Technology Stack

## 1. 技術スタック (Tech Stack)
バックエンドはGoで統一し、銘柄ごとのインメモリOrderBookワーカーを中心に構築します。低レイテンシと厳密な順序保証を最優先にします。

*   **Frontend**: HTML5 / JavaScript (Vanilla ES6+)
    *   ビルドツール不要のシンプルな構成。
    *   **CSS Framework**: Tailwind CSS または Bootstrap 5 でデザインを効率化。
    *   **Charting Library**: Lightweight Charts (TradingView) - 高性能かつ軽量なチャート描画。
*   **Backend**: Go 1.22+（HTTP API + WebSocket + Matching Engine）
    *   GoroutineとChannelを活用し、高頻度な注文処理を低遅延で処理。
    *   **API Specification**: OpenAPI (Swagger UI) を生成し、フロントエンド・Bot開発の効率化を図る。
*   **Database**: MySQL 8.0
    *   ユーザーデータ、資産、注文履歴の永続化。
*   **Infrastructure**: Docker & Docker Compose
    *   **App Container**: Goアプリケーションサーバー（API / WebSocket / Matching Engine）
    *   **DB Container**: MySQL
    *   **Bot Containers**: 複数のBotプロセス（Market Maker, Whale等）を独立コンテナとして起動。Docker内部ネットワーク経由でAPIサーバーに接続する。
*   **Real-time**: WebSocket（Goサーバー内の配信機構を利用）
*   **Authentication**: Discord OAuth2 + APIキー
    *   ユーザー認証はDiscordアカウントを使用。ログイン成功時に10バイトのAPIキーを発行し、20文字の16進数としてAPIリクエストの認証に使用する。


## 2. データベース設計と整合性 (Database Integrity)
金融シミュレーションにおいてデータの正確性は最優先事項です。

### 2.1. データ整合性の確保 (Data Integrity)
*   **統合ポジション管理 (Unified Positions)**: すべての保有資産（現物、信用）を `positions` テーブルで一元管理し、複雑な状態（現引き/現渡し）における不整合を防ぎます。
*   **整数管理 (Integer Math)**: 通貨や価格はすべて整数（例: 1.00ドル = 100セント）としてDBに格納し、浮動小数点演算による誤差（丸め誤差）を排除します。

### 2.2. 主要テーブルスキーマ (Schema Design)
主要なエンティティの設計案（抜粋）です。

*   **users**
    *   `id` (UUID): 内部識別子
    *   `discord_id` (VARCHAR): DiscordユーザーID（ユニーク制約）
    *   `username` (VARCHAR): 表示名
    *   `role` (ENUM): ADMIN, PLAYER, BOT
    *   `created_at` (DATETIME)

*   **assets** (銘柄マスタ)
    *   `symbol` (VARCHAR): ティッカーシンボル (e.g., "OMNI")
    *   `name` (VARCHAR): 銘柄名
    *   `type` (ENUM): STOCK, BOND, FOREX
    *   `sector` (VARCHAR): セクター

*   **orders** (注文)
    *   `id` (UUID): 注文ID
    *   `user_id` (UUID, FK): 発注者
    *   `symbol` (VARCHAR, FK): 銘柄
    *   `side` (ENUM): BUY, SELL
    *   `type` (ENUM): MARKET, LIMIT
    *   `price` (BIGINT): 指値価格（成行はNULLまたは0）
    *   `quantity` (BIGINT): 注文数量
    *   `status` (ENUM): OPEN, FILLED, CANCELED, REJECTED
    *   `filled_quantity` (BIGINT): 約定済み数量

*   **executions** (約定履歴)
    *   `id` (UUID): 約定ID
    *   `order_id` (UUID, FK): 紐づく注文
    *   `price` (BIGINT): 約定価格
    *   `quantity` (BIGINT): 約定数量
    *   `executed_at` (DATETIME): 約定日時

*   **positions** (保有ポジション)
    *   `user_id` (UUID, FK)
    *   `symbol` (VARCHAR, FK)
    *   `quantity` (BIGINT): 保有数量（正=ロング, 負=ショート）
    *   `average_price` (BIGINT): 平均取得単価


## 3. スケーラビリティとパフォーマンス (Scalability & Performance)
MMOとして多数の同時接続と注文処理に耐える設計を目指します。

### 3.1. 注文処理の同時実行性 (Concurrency)
*   **Engine Router**: `Engine` が銘柄ごとの `OrderBook` ワーカー（Goroutine）を管理し、注文を該当銘柄の `OrderChannel` にルーティングします。
*   **Single Writer Principle**: 1銘柄の板（Bids/Asks）を更新できるのは当該銘柄の1つのGoroutineのみとし、Mutexなしで厳密な順序保証を実現します。
*   **非同期永続化**: 約定結果・注文状態更新・ポジション更新は、DB書き込み専用Goroutine（ワーカープール）へ渡してMySQLへ非同期 `INSERT/UPDATE` します。
*   **WebSocket直結配信**: 約定と板更新をインメモリ状態から直接配信し、Redis Pub/Subを介しません。

### 3.2. 障害耐性 (Resilience)
*   **起動時リカバリ**: サーバー起動時に `orders` テーブルから `status = 'OPEN'` または `'PARTIAL'` の注文を全件ロードし、インメモリ板をウォームアップ再構築します。
*   **Graceful Shutdown**: 終了シグナル受信後は新規受付を停止し、DB書き込みキューの未処理イベントがゼロになるまで待機してから停止します。
*   **事前Hold**: 注文をチャネル投入する前に、`currency_balances` / `asset_balances` 上で必要資金・株数を拘束（Hold）し、整合性を担保します。

### 3.3. 将来的な拡張 (Future Proofing)
*   **DBシャーディング**: ユーザー数や取引量の増加に備え、`orders` や `executions` などのトランザクション系テーブルは、将来的にシャーディング（分割）可能な設計を意識します。
