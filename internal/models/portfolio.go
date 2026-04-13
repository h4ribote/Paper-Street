package models

type Balance struct {
	Currency     string `json:"currency"`
	Amount       int64  `json:"amount"`
	LockedAmount int64  `json:"locked_amount"`
}

type Position struct {
	AssetID        int64 `json:"asset_id"`
	Quantity       int64 `json:"quantity"`
	LockedQuantity int64 `json:"locked_quantity"`
}

type GetBalanceParams struct {
	UserID   int64
	Currency string
}

type GetPositionParams struct {
	UserID  int64
	AssetID int64
}
