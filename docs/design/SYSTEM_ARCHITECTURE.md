# System Architecture & Technology Stack

## 1. 技術スタック (Tech Stack)
バックエンドはPythonで統一し、Dockerで環境を構築します。拡張性とメンテナンス性を重視します。

*   **Frontend**: HTML5 / JavaScript (Vanilla ES6+)
    *   ビルドツール不要のシンプルな構成。
    *   **CSS Framework**: Tailwind CSS または Bootstrap 5 でデザインを効率化。
    *   **Charting Library**: Lightweight Charts (TradingView) - 高性能かつ軽量なチャート描画。
*   **Backend**: Python 3.12+ (FastAPI)
    *   非同期処理（`asyncio`）を活用し、高頻度な注文処理をさばく。
*   **Database**: MySQL 8.0
    *   ユーザーデータ、資産、注文履歴の永続化。
*   **Infrastructure**: Docker & Docker Compose
    *   **App Container**: FastAPIサーバー
    *   **DB Container**: MySQL
    *   **Bot Containers**: 複数のPythonスクリプト（Market Maker, Whale等）を独立したコンテナとして起動し、API経由で市場に参加させる。
*   **Real-time**: WebSocket (FastAPI標準のサポートを利用)
*   **Authentication**: Discord OAuth2

---

## 2. データベース設計と整合性 (Database Integrity)
金融シミュレーションにおいてデータの正確性は最優先事項です。

### 2.1. データ整合性の確保 (Data Integrity)
*   **統合ポジション管理 (Unified Positions)**: すべての保有資産（現物、信用）を `positions` テーブルで一元管理し、複雑な状態（現引き/現渡し）における不整合を防ぎます。
*   **整数管理 (Integer Math)**: 通貨や価格はすべて整数（例: 1.00ドル = 100セント）としてDBに格納し、浮動小数点演算による誤差（丸め誤差）を排除します。

---

## 3. スケーラビリティとパフォーマンス (Scalability & Performance)
MMOとして多数の同時接続と注文処理に耐える設計を目指します。

### 3.1. 注文処理の同時実行性 (Concurrency)
*   **Redisの活用**: 板情報（Order Book）の管理とマッチング処理は、MySQLではなくメモリ上のデータストア（Redis等）で行います。これにより、高速な約定処理とロック競合の回避を実現します。
*   **非同期永続化**: 約定結果のみを非同期でMySQLに書き込むアーキテクチャを採用し、DB負荷を軽減します。

### 3.2. 将来的な拡張 (Future Proofing)
*   **DBシャーディング**: ユーザー数や取引量の増加に備え、`orders` や `executions` などのトランザクション系テーブルは、将来的にシャーディング（分割）可能な設計を意識します。
