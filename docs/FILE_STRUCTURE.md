# ファイル構成

Paper Street の主要なディレクトリ構成と役割をまとめます。

## 概要

```
.
├── cmd/
│   └── paper-street-server/   # サーバー実行バイナリ
├── configs/                   # 環境変数・設定テンプレート
├── deployments/
│   ├── docker/                # Dockerfile 群
│   └── docker-compose.yml      # Compose 定義
├── docs/                       # ドキュメント
├── frontend/
│   ├── public/                 # 静的ファイル (HTML)
│   └── src/                    # JS/CSS などのフロント実装
├── internal/                   # アプリ内部ロジック
│   ├── api/                    # HTTP ハンドラ/ルーティング
│   ├── bots/                   # ボットのロジック/戦略
│   ├── db/                     # DB 接続/クエリ
│   ├── engine/                 # マッチングエンジン/板
│   ├── models/                 # ドメインモデル
│   └── websocket/              # WebSocket ハブ/クライアント
├── pkg/                        # 共有ユーティリティ
└── init.sql                    # 初期 DB セットアップ
```

## 補足

- `cmd/` は実行バイナリ単位でディレクトリを切り、バイナリ名がそのままパスになります。
- `configs/` は環境依存の設定テンプレートをまとめ、ルート直下の散在を防ぎます。
- `frontend/` は `public` と `src` に分割し、将来のビルドツール導入を前提に整理しています。
- `deployments/docker/` に Dockerfile を集約し、環境別の追加に備えます。
