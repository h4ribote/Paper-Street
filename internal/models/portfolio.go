package models

type Balance struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

type Position struct {
	AssetID  int64 `json:"asset_id"`
	Quantity int64 `json:"quantity"`
}

type GetBalanceParams struct {
	UserID   int64
	Currency string
}

type GetPositionParams struct {
	UserID  int64
	AssetID int64
}
