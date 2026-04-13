package bots

type Asset struct {
	ID     int64  `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Sector string `json:"sector"`
}

type Candle struct {
	Timestamp int64 `json:"timestamp"`
	Open      int64 `json:"open"`
	High      int64 `json:"high"`
	Low       int64 `json:"low"`
	Close     int64 `json:"close"`
	Volume    int64 `json:"volume"`
}

type MacroIndicator struct {
	Country     string `json:"country"`
	Type        string `json:"type"`
	Value       int64  `json:"value"`
	PublishedAt int64  `json:"published_at"`
}

type TheoreticalFXRate struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Rate          int64  `json:"rate"`
	UpdatedAt     int64  `json:"updated_at"`
}

type PerpetualBondInfo struct {
	Asset            Asset  `json:"asset"`
	IssuerCountry    string `json:"issuer_country"`
	Currency         string `json:"currency"`
	BaseCoupon       int64  `json:"base_coupon"`
	PaymentFrequency string `json:"payment_frequency"`
	TargetYieldBps   int64  `json:"target_yield_bps"`
	TheoreticalPrice int64  `json:"theoretical_price"`
}

type LiquidityPool struct {
	ID            int64  `json:"id"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	FeeBps        int64  `json:"fee_bps"`
	Liquidity     int64  `json:"liquidity"`
	CurrentTick   int64  `json:"current_tick"`
}

type PoolSwapRequest struct {
	UserID       int64  `json:"user_id"`
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	Amount       int64  `json:"amount"`
}

type PoolSwapResult struct {
	PoolID       int64  `json:"pool_id"`
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	AmountIn     int64  `json:"amount_in"`
	AmountOut    int64  `json:"amount_out"`
	FeeAmount    int64  `json:"fee_amount"`
}

type IndexActionRequest struct {
	UserID   int64 `json:"user_id"`
	Quantity int64 `json:"quantity"`
}

type IndexActionResult struct {
	AssetID     int64   `json:"asset_id"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   int64   `json:"unit_price"`
	FeeAmount   int64   `json:"fee_amount"`
	TotalAmount int64   `json:"total_amount"`
	Components  []int64 `json:"components"`
	Action      string  `json:"action"`
}

type Balance struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

type PortfolioAsset struct {
	Asset        Asset `json:"asset"`
	Quantity     int64 `json:"quantity"`
	AveragePrice int64 `json:"average_price,omitempty"`
}

type MarginPosition struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	BaseAssetID  int64  `json:"base_asset_id,omitempty"`
	AssetID      int64  `json:"asset_id,omitempty"`
	Side         string `json:"side"`
	Quantity     int64  `json:"quantity"`
	EntryPrice   int64  `json:"entry_price"`
	Leverage     int    `json:"leverage,omitempty"`
	LiquidationP int64  `json:"liquidation_price,omitempty"`
}
