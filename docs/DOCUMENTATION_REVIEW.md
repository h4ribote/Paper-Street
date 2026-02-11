# ドキュメントレビュー結果

本プロジェクトのドキュメント（`docs/`ディレクトリ配下）を精査し、以下の不完全な部分、不整合、および明確化が必要な項目を特定しました。
すべての項目について、**2025/05/XX (Updated)** に対応が完了しています。

## 1. SYSTEM_ARCHITECTURE.md

### 不整合: Redisの記述漏れ (Resolved)
*   **箇所**: `3.1. 注文処理の同時実行性`
*   **問題**: 「板情報（Order Book）の管理とマッチング処理は、MySQLではなくメモリ上のデータストア（Redis等）で行います」と記述されていますが、`1. 技術スタック` のリストには Redis が記載されていません。
*   **対応**: 技術スタックに Redis 7.0 (Order Book, Pub/Sub, Session Store) を追加しました。

### 不足: ユーザー認証・管理の詳細 (Resolved)
*   **箇所**: `1. 技術スタック` - Authentication
*   **問題**: "Discord OAuth2" とありますが、ユーザーセッションの管理方法（JWT, Session Cookies?）や、ユーザープロファイル（DB上の `users` テーブル構造など）に関する記述が欠落しています。
*   **対応**: Authentication セクションを詳細化し、JWT と `users` テーブルスキーマを `SYSTEM_ARCHITECTURE.md` に追加しました。

### 不足: Botコンテナのオーケストレーション (Resolved)
*   **箇所**: `1. 技術スタック` - Infrastructure
*   **問題**: "Bot Containers" が API 経由で市場に参加するとありますが、具体的な接続方法（内部ネットワーク経由か、公開APIか）や、デプロイメント構成（docker-compose上での定義など）が不明確です。
*   **対応**: Docker内部ネットワーク経由での接続を明記しました。

---

## 2. ECONOMY.md & GAMEPLAY_AND_UI.md

### 未定（TBD/Planned）機能の散在 (Resolved)
以下の機能は「予定」「将来的な拡張」とされており、現時点での仕様が未確定です。これらがMVP（Minimum Viable Product）に含まれるかどうかの明確化が必要です。

*   **債券発行**: プレイヤー自身による債券発行機能（`ECONOMY.md` 2. 資産クラス / `GAMEPLAY_AND_UI.md` 4.1）。
*   **デリバティブ**: オプションや先物取引（`ECONOMY.md` 2. 資産クラス）。
*   **Maker手数料リベート**: 指値注文による手数料リベート（`GAMEPLAY_AND_UI.md` 4.1）。

*   **対応**: すべて `(Post-MVP)` として明記し、初期リリースには含まれないことを明確にしました。

### FX市場の仕様詳細 (Resolved)
*   **箇所**: `ECONOMY.md` 1.1 および `GAMEPLAY_AND_UI.md` 4.1
*   **問題**: FX市場は「流動性プール」を使用するとありますが、具体的な価格決定アルゴリズム（AMM: Automated Market Makerのようなものか？）や、プールへの流動性提供時の報酬計算式が記述されていません。また、指値注文が「指定価格帯への集中流動性提供」として扱われる際のUX詳細も不足しています。
*   **対応**: 定数積モデル ($x \times y = k$) を採用する旨と、指値機能の Post-MVP 化を記述しました。

---

## 3. AI_ECOSYSTEM.md

### 学習型AIの具体性 (Resolved)
*   **箇所**: `2.2. 学習型AIと適応`
*   **問題**: 「過去の損失パターンの回避」や「噂と事実の区別」といった高度な振る舞いが記述されていますが、これを実現するための技術的アプローチ（強化学習？ ルールベース？）やデータ構造についての言及がありません。実装難易度が高いため、初期フェーズでの実現可能性を再評価する必要があります。
*   **対応**: 初期フェーズでは「高度なルールベースと状態遷移」で実装し、強化学習は将来的な拡張とすることを明記しました。

### 国家AIとの連携 (Resolved)
*   **箇所**: `ECONOMY.md` 6. 国家の市場介入
*   **問題**: `ECONOMY.md` で言及されている「国家AI（National AI）」が、`AI_ECOSYSTEM.md` のボット分類（Market Maker, Whale等）に含まれていません。国家AIがどのようなアルゴリズム（Botタイプ）として実装されるかの定義が必要です。
*   **対応**: `AI_ECOSYSTEM.md` に "National AI" ロールを追加しました。

---

## 4. 全体的な不足事項

### API仕様書の欠落 (Resolved)
*   **問題**: フロントエンドとバックエンド（およびBot）が通信するための具体的な API エンドポイント定義（OpenAPI/Swagger等）が存在しません。
*   **対応**: `SYSTEM_ARCHITECTURE.md` に OpenAPI (Swagger) の使用を追記しました。

### データベーススキーマ定義 (Resolved)
*   **問題**: `SYSTEM_ARCHITECTURE.md` に概念的な記述はありますが、具体的なテーブル定義（ER図やDDL）が含まれていません。特に `positions`, `orders`, `executions`, `users` などの主要テーブルのスキーマ設計が必要です。
*   **対応**: `SYSTEM_ARCHITECTURE.md` に主要テーブルのスキーマ設計を追記しました。
