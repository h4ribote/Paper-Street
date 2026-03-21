package engine

import "time"

type Execution struct {
	ID            int64     `json:"id"`
	AssetID       int64     `json:"asset_id"`
	Price         int64     `json:"price"`
	Quantity      int64     `json:"quantity"`
	TakerOrderID  int64     `json:"taker_order_id"`
	MakerOrderID  int64     `json:"maker_order_id"`
	TakerUserID   int64     `json:"taker_user_id"`
	MakerUserID   int64     `json:"maker_user_id"`
	OccurredAtUTC time.Time `json:"occurred_at"`
}

func newExecution(assetID int64, price int64, quantity int64, taker *Order, maker *Order) Execution {
	return Execution{
		AssetID:       assetID,
		Price:         price,
		Quantity:      quantity,
		TakerOrderID:  taker.ID,
		MakerOrderID:  maker.ID,
		TakerUserID:   taker.UserID,
		MakerUserID:   maker.UserID,
		OccurredAtUTC: time.Now().UTC(),
	}
}
