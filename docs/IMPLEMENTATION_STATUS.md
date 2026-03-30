# バックエンド実装状況メモ

本ドキュメントは、`docs/design/` に記載された仕様のうち **バックエンド側で設計のみ / 未実装 / 部分実装** のものを整理したメモです。
実装の実態は `/internal/` 配下のコードを基準に確認しています。

## 未実装・部分実装の仕様一覧

| 設計ドキュメント | 仕様の要点 | 現状のバックエンド実装 | 状態 |
| --- | --- | --- | --- |
| [AI_ECOSYSTEM.md](./design/AI_ECOSYSTEM.md) | 12種類のボット戦略 | `internal/bots/` と `cmd/bots/` にあるのは **Market Maker / News Reactor のみ**。他の戦略は未実装。 | 未実装 |
| [THEORETICAL_FX_RATE.md](./design/THEORETICAL_FX_RATE.md) | マクロ指標から理論FXレート算出 | 該当する計算・保存・配信処理は見当たらない。 | 未実装 |
| [INDICES.md](./design/INDICES.md) | バスケット連動のCreation/Redemption・FX換算 | `internal/api/liquidity.go` に **単純合計価格の指数**とCreate/Redeemはあるが、**構成銘柄バスケットの受渡・FX換算・裁定バンド**は未実装。 | 部分実装 |
