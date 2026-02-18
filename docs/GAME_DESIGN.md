# Game Design Document: Paper Street

**Real-time Financial MMO Simulation**

## 1. プロジェクト概要
**Paper Street** は、現実的な金融市場をシミュレートしたトレーディングゲームです。「Wall Street Junior」をインスピア元とし、プロフェッショナルなトレーダーとしての体験を提供します。

本ドキュメントは、プロジェクトの詳細な仕様を記述するインデックスです。各機能の詳細は、以下のセクション別ドキュメントを参照してください。

## 2. デザイン詳細 (Detailed Design Specs)

### [Core Gameplay & UI/UX](./design/GAMEPLAY_AND_UI.md)
*   **コアゲームループ**: 情報収集、分析、取引実行、リスク管理のサイクル。
*   **モチベーションとリテンション**: デイリーミッション、ログインボーナス、破産時の救済措置（ベーシックインカム）。
*   **The Terminal UI**: プロフェッショナルなトレーダーが使用する情報端末の画面レイアウトと機能。
    UIおよびニュースコンテンツは**日本語と英語**に対応しており、プレイヤーは設定で切り替えが可能です。
*   **プレイヤーアクション**: 注文タイプ（成行、指値、逆指値）。

### [Trader Rank & Progression](./design/TRADER_RANK_SYSTEM.md)
*   **ランクシステム**: ShrimpからLeviathanまでの5段階。手数料と金利の優遇。
*   **XP獲得**: デイリーミッションの完遂によるXP獲得とランクアップ。
*   **シーズン移行**: 前シーズンのランクとランキングに応じた「強くてニューゲーム」ボーナス。

### [Economy & Market System](./design/ECONOMY.md)
*   **高度な市場シミュレーション**: 24時間市場、板情報、流動性、マクロ経済指標。
*   **資産クラス**: 株式、債券、FX、デリバティブ。
*   **シーズン制**: ランキング、報酬、シーズンごとのテーマ（大恐慌、バブル等）。
*   **Dual Liquidity Inventory (DLI)**: [詳細設計](./design/DUAL_LIQUIDITY_INVENTORY.md) - 現金と現物の在庫を利用率に応じて管理する、金利とショートコストの動的決定メカニズム。

### [AI Traders & Bot Ecosystem](./design/AI_ECOSYSTEM.md)
*   **ボットの役割**: Market Maker, Trend Follower, Mean Reverter, Whale, News Reactor。
*   **AIの挙動改善**: パラメータのランダム化、反応遅延（Jitter）、学習型AIによる対抗策。

### [System Architecture & Tech Stack](./design/SYSTEM_ARCHITECTURE.md)
*   **技術スタック**: Python (FastAPI), MySQL, Docker, Lightweight Charts。
*   **データベース設計**: 統合ポジション管理、整数演算によるデータ整合性の確保。
*   **スケーラビリティ**: Redisを活用した高速な注文処理（板情報管理）、非同期永続化。

---

## 3. 世界観・設定資料 (Lore & Reference)

### [World Setting & Lore](./world_setting.md)
*   架空の国家（Neo Venice, Boros Federation等）、通貨、企業、イベント設定。

### [Wall Street Junior Reference](./WALL_STREET_JUNIOR_REFERENCE.md)
*   インスピレーション元となったゲームの調査報告書。
