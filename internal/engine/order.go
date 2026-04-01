package engine

import "time"

type Side string

type OrderType string

type OrderStatus string

type TimeInForce string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

const (
	OrderTypeLimit     OrderType = "LIMIT"
	OrderTypeMarket    OrderType = "MARKET"
	OrderTypeStop      OrderType = "STOP"
	OrderTypeStopLimit OrderType = "STOP_LIMIT"
)

const (
	OrderStatusOpen      OrderStatus = "OPEN"
	OrderStatusPartial   OrderStatus = "PARTIAL"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
)

const (
	TimeInForceGTC TimeInForce = "GTC"
	TimeInForceIOC TimeInForce = "IOC"
	TimeInForceFOK TimeInForce = "FOK"
)

type Order struct {
	ID          int64       `json:"id"`
	UserID      int64       `json:"user_id"`
	AssetID     int64       `json:"asset_id"`
	Side        Side        `json:"side"`
	Type        OrderType   `json:"type"`
	TimeInForce TimeInForce `json:"time_in_force,omitempty"`
	Quantity    int64       `json:"quantity"`
	Remaining   int64       `json:"remaining"`
	Price       int64       `json:"price,omitempty"`
	StopPrice   int64       `json:"stop_price,omitempty"`
	Leverage    int64       `json:"leverage,omitempty"`
	Status      OrderStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (o *Order) clone() *Order {
	copy := *o
	return &copy
}

func (o *Order) isActive() bool {
	return o.Status == OrderStatusOpen || o.Status == OrderStatusPartial
}

func (o *Order) isStopOrder() bool {
	return o.Type == OrderTypeStop || o.Type == OrderTypeStopLimit
}

func (o *Order) effectiveTimeInForce() TimeInForce {
	if o == nil || o.TimeInForce == "" {
		return TimeInForceGTC
	}
	return o.TimeInForce
}

func (o *Order) fill(quantity int64) {
	o.Remaining -= quantity
	o.UpdatedAt = time.Now().UTC()
	if o.Remaining <= 0 {
		o.Remaining = 0
		o.Status = OrderStatusFilled
		return
	}
	o.Status = OrderStatusPartial
}

func (o *Order) cancel() {
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now().UTC()
}

func (o *Order) reject() {
	o.Status = OrderStatusRejected
	o.UpdatedAt = time.Now().UTC()
}

func (o *Order) open() {
	if o.Remaining <= 0 {
		o.Remaining = o.Quantity
	}
	o.Status = OrderStatusOpen
	o.UpdatedAt = time.Now().UTC()
}
