# バックエンド実装状況メモ

本ドキュメントは、`docs/design/` に記載された仕様のうち **バックエンド側で設計のみ / 未実装 / 部分実装** のものを整理したメモです。
実装の実態は `/internal/` 配下のコードを基準に確認しています。

## 未実装・部分実装の仕様一覧

| 設計ドキュメント | 仕様の要点 | 現状のバックエンド実装 | 状態 |
| --- | --- | --- | --- |
| [AI_ECOSYSTEM.md](./design/AI_ECOSYSTEM.md) | 12種類のボット戦略 | `internal/bots/` と `cmd/bots/` にあるのは **Market Maker / News Reactor のみ**。他の戦略は未実装。 | 未実装 |
| [INDICES.md](./design/INDICES.md) | バスケット連動のCreation/Redemption・FX換算 | `internal/api/liquidity.go` に **単純合計価格の指数**とCreate/Redeemはあるが、**構成銘柄バスケットの受渡・FX換算・裁定バンド**は未実装。 | 部分実装 |
| [EQUITY_FINANCING.md](./design/EQUITY_FINANCING.md) | 企業の資金調達・自社株買い（Treasury Stock Sale / New Issuance / Buyback） | 企業の資本構成（発行済/流通/自己株）、資金調達・買い戻しロジック、専用API（capital-structure/financing/buyback）を実装済。 | 実装済 |
| [ECONOMIC_SIMULATION_DETAILS.md](./design/ECONOMIC_SIMULATION_DETAILS.md) | 企業Botの生産・調達・在庫・価格サイクル | 生産レシピ/原材料を使った調達・生産・販売・決算ロジックと、production-status/supply-chain/financials/simulate APIを実装済。 | 実装済 |
| [ECONOMY.md](./design/ECONOMY.md) | 永久債（Consol）の発行・買い戻し・クーポン支払い | `perpetual_bonds` テーブル以外の処理（発行オペ、利払いバッチ、価格ロジック）が未実装。 | 未実装 |
| [NEWS_SYSTEM.md](./design/NEWS_SYSTEM.md) | 定期/ランダムニュース生成とNews Reactorの自動売買 | サーバーは起動時の `seedNews` のみで、定期生成・センチメントに基づく自動注文/市場インパクト処理が見当たらない。 | 部分実装 |
| [MACRO_ECONOMICS.md](./design/MACRO_ECONOMICS.md) | GDP/CPI/失業率/金利を実需データ（C+I+G+X-M等）から算出 | 指標生成はあるが、経済活動データに連動した算出ロジックは未実装。 | 部分実装 |
