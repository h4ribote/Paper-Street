package models

type Balance struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

type Position struct {
	AssetID  int64 `json:"asset_id"`
	Quantity int64 `json:"quantity"`
}
