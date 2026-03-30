# バックエンド実装状況メモ

本ドキュメントは、`docs/design/` に記載された仕様のうち **バックエンド側で設計のみ / 未実装 / 部分実装** のものを整理したメモです。
実装の実態は `/internal/` 配下のコードを基準に確認しています。

## 未実装・部分実装の仕様一覧

| 設計ドキュメント | 仕様の要点 | 現状のバックエンド実装 | 状態 |
| --- | --- | --- | --- |
| [AI_ECOSYSTEM.md](./design/AI_ECOSYSTEM.md) | 12種類のボット戦略 | `internal/bots/` と `cmd/bots/` にあるのは **Market Maker / News Reactor のみ**。他の戦略は未実装。 | 未実装 |
| [DUAL_LIQUIDITY_INVENTORY.md](./design/DUAL_LIQUIDITY_INVENTORY.md) | 利用率ベース金利・金利徴収サイクル・借入フロー | `marginRates` に利用率のキンク計算はあるが、**借入/返済フローや定期金利徴収の処理が未実装**。 | 部分実装 |
| [CONTRACTS.md](./design/CONTRACTS.md) | 経済状況に応じた契約生成・VWAP + プレミアム価格 | `seedContracts` で **固定2件をシード**。動的な発行ロジックや市場連動の価格計算は未実装。 | 部分実装 |
| [MACRO_ECONOMICS.md](./design/MACRO_ECONOMICS.md) | GDP/CPI/失業率などの算出・更新 | `internal/api/store.go` で **定数の指標をシード**しているのみ。計算・更新ロジックは未実装。 | 未実装 |
| [THEORETICAL_FX_RATE.md](./design/THEORETICAL_FX_RATE.md) | マクロ指標から理論FXレート算出 | 該当する計算・保存・配信処理は見当たらない。 | 未実装 |
| [INDICES.md](./design/INDICES.md) | バスケット連動のCreation/Redemption・FX換算 | `internal/api/liquidity.go` に **単純合計価格の指数**とCreate/Redeemはあるが、**構成銘柄バスケットの受渡・FX換算・裁定バンド**は未実装。 | 部分実装 |
