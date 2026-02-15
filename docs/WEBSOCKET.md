# WebSocket API Specification

Paper Street のリアルタイムデータ配信のためのWebSocket API仕様書です。

## 1. 接続情報 (Connection)
*   **Endpoint**: `ws://api.paperstreet.com/ws` (または `wss://` for SSL)
*   **Authentication**:
    *   クエリパラメータ `token` にJWTトークンを含めて接続します。
    *   例: `ws://api.paperstreet.com/ws?token=eyJhbGciOi...`
    *   認証に失敗した場合、接続は即座に切断されます (Close Code: 4001)。

## 2. メッセージフォーマット (Message Format)
クライアント・サーバー間の通信はすべて JSON フォーマットで行います。

### リクエスト (Client -> Server)
購読 (Subscribe) や購読解除 (Unsubscribe) を行います。

```json
{
  "op": "subscribe",
  "args": ["market.ticker", "orderbook.101"]
}
```

*   `op`: 操作タイプ (`subscribe`, `unsubscribe`, `ping`)
*   `args`: チャンネル名のリスト

### レスポンス (Server -> Client)
データ更新やエラー通知が配信されます。

```json
{
  "topic": "market.ticker",
  "data": {
    "symbol": "OMNI",
    "price": 15025,
    "change": 25,
    "volume": 10500
  },
  "ts": 1678892345000
}
```

*   `topic`: チャンネル名
*   `data`: ペイロード (内容はチャンネルにより異なる)
*   `ts`: タイムスタンプ (ミリ秒)

## 3. チャンネル一覧 (Channels)

### パブリックチャンネル (Public)
認証済みユーザーであれば誰でも購読可能です。

*   `market.ticker`
    *   全銘柄の現在値、変動率などのサマリー情報を配信します（1秒ごとのスナップショット）。
*   `market.orderbook.{asset_id}`
    *   指定した銘柄の板情報（深さ20）の更新を配信します。
    *   初回購読時にスナップショットが送信され、以降は差分更新 (Delta) が配信されます。
*   `market.trade.{asset_id}`
    *   指定した銘柄の約定（歩み値）をリアルタイムで配信します。
*   `market.candles.{asset_id}.{timeframe}`
    *   ローソク足の更新を配信します（足が確定したタイミング、またはリアルタイム）。
*   `news`
    *   ニュースヘッドラインの速報を配信します。

### プライベートチャンネル (Private)
接続しているユーザー自身のデータのみが配信されます。

*   `user.orders`
    *   自分の注文ステータス変更（部分約定、約定、キャンセル）を配信します。
*   `user.executions`
    *   自分の約定レポートを配信します。
*   `user.portfolio`
    *   資産残高やポジション評価損益の変動を配信します。

## 4. エラーコード (Error Codes)
*   `4000`: 正常終了
*   `4001`: 認証エラー (Invalid Token)
*   `4002`: 無効なメッセージ形式
*   `4003`: 購読制限超過 (Too many subscriptions)
*   `5000`: サーバー内部エラー
