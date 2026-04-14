package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	defaultRegionName   = "Arcadia"
	defaultCountryName  = "Arcadia"
	defaultSectorName   = "UNKNOWN"
	defaultCurrencyName = "Arcadian Credit"
	defaultUserRankID   = 1
	defaultAssetPrice   = int64(10_000)
)

type Queries struct {
	Conn *Connection
}

type AssetSnapshot struct {
	Asset     models.Asset
	BasePrice int64
}

type CurrencyBalance struct {
	UserID       int64
	Currency     string
	Amount       int64
	LockedAmount int64
}

type AssetBalance struct {
	UserID         int64
	AssetID        int64
	Quantity       int64
	LockedQuantity int64
}

type APIKeyRecord struct {
	Key    string
	UserID int64
	Role   string
}

type PerpetualBondRecord struct {
	AssetID          int64
	IssuerCountry    string
	BaseCoupon       int64
	PaymentFrequency string
}

type IndexConstituentRecord struct {
	IndexAssetID     int64
	ComponentAssetID int64
}

type ExecutionRecord struct {
	ID           int64
	AssetID      int64
	Price        int64
	Quantity     int64
	ExecutedAt   time.Time
	BuyOrderID   int64
	SellOrderID  int64
	IsTakerBuyer bool
}

type NewsRecord struct {
	ID             int64
	Headline       string
	Body           string
	PublishedAt    int64
	Source         string
	SentimentScore int64
	AssetID        int64
	Category       string
	Impact         string
	ImpactScope    string
}

type CompanyRecord struct {
	CompanyID             int64
	Name                  string
	Symbol                string
	Sector                string
	Country               string
	UserID                sql.NullInt64
	MaxProductionCapacity int64
	CurrentInventory      int64
	LastCapexAt           int64
	SharesIssued          int64
	SharesOutstanding     int64
	TreasuryStock         int64
}

type ProductionRecipeRecord struct {
	ID             int64
	CompanyID      int64
	OutputAssetID  int64
	OutputQuantity int64
}

type ProductionInputRecord struct {
	ID            int64
	RecipeID      int64
	InputAssetID  int64
	InputQuantity int64
}

type FinancialReportRecord struct {
	CompanyID       int64
	FiscalYear      int
	FiscalQuarter   int
	Revenue         int64
	NetIncome       int64
	EPS             int64
	Capex           int64
	UtilizationRate int64
	InventoryLevel  int64
	Guidance        string
	PublishedAt     int64
}

type CompanyDividendRecord struct {
	CompanyID          int64
	AssetID            int64
	FiscalYear         int
	FiscalQuarter      int
	NetIncome          int64
	PayoutRatioBps     int64
	DividendPerShare   int64
	CompanyPayout      int64
	PoolPayout         int64
	SpotPayout         int64
	MarginLongPayout   int64
	MarginShortCharge  int64
	EligibleSpotShares int64
	EligibleLongShares int64
	PoolShares         int64
	CreatedAt          int64
}

type LiquidityPoolRecord struct {
	PoolID         int64
	QuoteCurrency  string
	FeeBps         int64
	CurrentTick    int64
	Liquidity      int64
	CreatedAt      int64
	TickSpacing    int64
	FeeGrowthBase  int64
	FeeGrowthQuote int64
}

type LiquidityPositionRecord struct {
	ID          int64
	PoolID      int64
	UserID      int64
	LowerTick   int64
	UpperTick   int64
	BaseAmount  int64
	QuoteAmount int64
	Liquidity   int64
	CreatedAt   int64
	UpdatedAt   int64
}

type MarginPoolRecord struct {
	PoolID           int64
	AssetID          int64
	Currency         string
	TotalCash        int64
	BorrowedCash     int64
	TotalAssets      int64
	BorrowedAssets   int64
	CashRateBps      int64
	AssetRateBps     int64
	TotalCashShares  int64
	TotalAssetShares int64
	UpdatedAt        int64
}

type MarginPoolProviderRecord struct {
	ID          int64
	PoolID      int64
	UserID      int64
	CashShares  int64
	AssetShares int64
	UpdatedAt   int64
}

type MarginPositionRecord struct {
	ID           int64
	UserID       int64
	AssetID      int64
	Side         string
	Quantity     int64
	EntryPrice   int64
	CurrentPrice int64
	Leverage     int64
	MarginUsed   int64
	UnrealizedPL int64
	CreatedAt    int64
	UpdatedAt    int64
}

type ContractRecord struct {
	ID            int64
	Title         string
	Description   string
	AssetID       int64
	TotalRequired int64
	Delivered     int64
	UnitPrice     int64
	XPPerUnit     int64
	MinRank       string
	Status        string
	StartAt       int64
	ExpiresAt     int64
}

type ContractDeliveryRecord struct {
	ID           int64
	ContractID   int64
	UserID       int64
	Quantity     int64
	PayoutAmount int64
	XPGained     int64
	DeliveredAt  int64
}

type SeasonRecord struct {
	ID       int64
	Name     string
	Theme    string
	StartAt  int64
	EndAt    int64
	IsActive bool
}

type RegionRecord struct {
	ID          int64
	Name        string
	Description string
}

type WorldEventRecord struct {
	ID          int64
	Name        string
	Description string
	StartsAt    int64
	EndsAt      int64
}

type MacroIndicatorRecord struct {
	Country     string
	Type        string
	Value       int64
	PublishedAt int64
}

func NewQueries(conn *Connection) *Queries {
	return &Queries{Conn: conn}
}

func (q *Queries) Close() error {
	if q == nil || q.Conn == nil {
		return nil
	}
	return q.Conn.Close()
}

func (q *Queries) EnsureDefaultCurrency(ctx context.Context, code string) (int64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, errors.New("currency code required")
	}
	var currencyID int64
	err := q.Conn.DB.QueryRowContext(ctx, "SELECT currency_id FROM currencies WHERE code = ? LIMIT 1", code).Scan(&currencyID)
	if err == nil {
		return currencyID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	tx, err := q.Conn.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	regionID, err := ensureRegion(ctx, tx, defaultRegionName)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	countryID, err := ensureCountry(ctx, tx, defaultCountryName, regionID)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO currencies (country_id, code, name) VALUES (?, ?, ?)", countryID, code, defaultCurrencyName)
	if err != nil {
		_ = tx.Rollback()
		currencyID, lookupErr := q.lookupCurrencyID(ctx, code)
		if lookupErr == nil {
			return currencyID, nil
		}
		return 0, err
	}
	currencyID, err = result.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return currencyID, nil
}

func (q *Queries) lookupCurrencyID(ctx context.Context, code string) (int64, error) {
	var currencyID int64
	err := q.Conn.DB.QueryRowContext(ctx, "SELECT currency_id FROM currencies WHERE code = ? LIMIT 1", code).Scan(&currencyID)
	if err != nil {
		return 0, err
	}
	return currencyID, nil
}

func (q *Queries) UpsertUser(ctx context.Context, user models.User, createdAt time.Time) error {
	if user.ID == 0 {
		return errors.New("user id required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	rankID := user.RankID
	if rankID <= 0 {
		rankID = defaultUserRankID
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO users (user_id, username, rank_id, created_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE username = VALUES(username), rank_id = VALUES(rank_id)
	`, user.ID, strings.TrimSpace(user.Username), rankID, createdAt.UnixMilli())
	return err
}

func (q *Queries) UpdateUserXP(ctx context.Context, userID int64, xp int64, rankID int64) error {
	_, err := q.Conn.DB.ExecContext(ctx, "UPDATE users SET xp = ?, rank_id = ? WHERE user_id = ?", xp, rankID, userID)
	return err
}

func (q *Queries) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT u.user_id, u.username, u.rank_id, rd.name
		FROM users u
		LEFT JOIN rank_definitions rd ON rd.rank_id = u.rank_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var user models.User
		var rankName sql.NullString
		if err := rows.Scan(&user.ID, &user.Username, &user.RankID, &rankName); err != nil {
			return nil, err
		}
		user.Rank = rankName.String
		user.Role = "player"
		users = append(users, user)
	}
	return users, rows.Err()
}

func (q *Queries) GetUser(ctx context.Context, userID int64) (models.User, error) {
	var user models.User
	var rankName sql.NullString
	err := q.Conn.DB.QueryRowContext(ctx, `
		SELECT u.user_id, u.username, u.rank_id, rd.name
		FROM users u
		LEFT JOIN rank_definitions rd ON rd.rank_id = u.rank_id
		WHERE u.user_id = ?
	`, userID).Scan(&user.ID, &user.Username, &user.RankID, &rankName)
	if err == nil {
		user.Rank = rankName.String
		user.Role = "player"
	}
	return user, err
}

func (q *Queries) UpsertAsset(ctx context.Context, asset models.Asset, basePrice int64) error {
	if asset.ID == 0 {
		return errors.New("asset id required")
	}
	if asset.Symbol == "" {
		if strings.TrimSpace(asset.Name) != "" {
			asset.Symbol = strings.ToUpper(strings.TrimSpace(asset.Name))
		}
	}
	asset.Symbol = strings.TrimSpace(asset.Symbol)
	if asset.Symbol == "" {
		return errors.New("asset symbol required")
	}
	if asset.Type == "" {
		asset.Type = "STOCK"
	}
	createdAt := time.Now().UTC().UnixMilli()
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO assets (asset_id, ticker, type, base_price, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE ticker = VALUES(ticker), type = VALUES(type), base_price = VALUES(base_price)
	`, asset.ID, asset.Symbol, strings.TrimSpace(asset.Type), basePrice, createdAt)
	return err
}

func (q *Queries) GetAssetPrice(ctx context.Context, assetID int64) (int64, error) {
	var price int64
	err := q.Conn.DB.QueryRowContext(ctx, "SELECT base_price FROM assets WHERE asset_id = ?", assetID).Scan(&price)
	return price, err
}

func (q *Queries) ListAssets(ctx context.Context) ([]AssetSnapshot, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, "SELECT asset_id, ticker, type, base_price FROM assets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var assets []AssetSnapshot
	for rows.Next() {
		var asset models.Asset
		var basePrice int64
		if err := rows.Scan(&asset.ID, &asset.Symbol, &asset.Type, &basePrice); err != nil {
			return nil, err
		}
		if asset.Name == "" {
			asset.Name = asset.Symbol
		}
		assets = append(assets, AssetSnapshot{Asset: asset, BasePrice: basePrice})
	}
	return assets, rows.Err()
}

func (q *Queries) GetAsset(ctx context.Context, assetID int64) (models.Asset, error) {
	var asset models.Asset
	err := q.Conn.DB.QueryRowContext(ctx, "SELECT asset_id, ticker, type FROM assets WHERE asset_id = ?", assetID).Scan(&asset.ID, &asset.Symbol, &asset.Type)
	if err == nil {
		if asset.Name == "" {
			asset.Name = asset.Symbol
		}
	}
	return asset, err
}

func (q *Queries) ListPerpetualBonds(ctx context.Context) ([]PerpetualBondRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT pb.asset_id, c.name, pb.base_coupon, pb.payment_frequency
		FROM perpetual_bonds pb
		JOIN countries c ON c.country_id = pb.issuer_country_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bonds []PerpetualBondRecord
	for rows.Next() {
		var record PerpetualBondRecord
		if err := rows.Scan(&record.AssetID, &record.IssuerCountry, &record.BaseCoupon, &record.PaymentFrequency); err != nil {
			return nil, err
		}
		bonds = append(bonds, record)
	}
	return bonds, rows.Err()
}

func (q *Queries) UpsertPerpetualBond(ctx context.Context, record PerpetualBondRecord) error {
	if record.AssetID == 0 {
		return errors.New("asset id required")
	}
	if strings.TrimSpace(record.IssuerCountry) == "" {
		return errors.New("issuer country required")
	}
	tx, err := q.Conn.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	regionID, err := ensureRegion(ctx, tx, defaultRegionName)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	countryID, err := ensureCountry(ctx, tx, record.IssuerCountry, regionID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	frequency := strings.ToUpper(strings.TrimSpace(record.PaymentFrequency))
	if frequency == "" {
		frequency = "WEEKLY"
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO perpetual_bonds (asset_id, issuer_country_id, base_coupon, payment_frequency)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE issuer_country_id = VALUES(issuer_country_id),
			base_coupon = VALUES(base_coupon),
			payment_frequency = VALUES(payment_frequency)
	`, record.AssetID, countryID, record.BaseCoupon, frequency)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (q *Queries) UpsertOrder(ctx context.Context, order *engine.Order) error {
	if order == nil {
		return errors.New("order required")
	}
	createdAt := order.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := order.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	filled := order.Quantity - order.Remaining
	if filled < 0 {
		filled = 0
	}
	res, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO orders (order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at)
		VALUES (NULLIF(?, 0), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE user_id = VALUES(user_id),
			asset_id = VALUES(asset_id),
			side = VALUES(side),
			type = VALUES(type),
			quantity = VALUES(quantity),
			price = VALUES(price),
			stop_price = VALUES(stop_price),
			filled_quantity = VALUES(filled_quantity),
			status = VALUES(status),
			updated_at = VALUES(updated_at)
	`, order.ID, order.UserID, order.AssetID, order.Side, order.Type, order.Quantity, order.Price, order.StopPrice, filled, order.Status, createdAt.UnixMilli(), updatedAt.UnixMilli())
	if err != nil {
		return err
	}
	if order.ID == 0 {
		id, err := res.LastInsertId()
		if err == nil && id > 0 {
			order.ID = id
		}
	}
	return nil
}

func (q *Queries) GetCrossingOrders(ctx context.Context, tx *sql.Tx, assetID int64, side engine.Side, price int64, isMarket bool) ([]*engine.Order, error) {
	var query string
	var args []interface{}
	if side == engine.SideBuy {
		if isMarket {
			query = `SELECT order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at
					 FROM orders
					 WHERE asset_id = ? AND side = 'SELL' AND status IN ('OPEN', 'PARTIAL')
					 ORDER BY price ASC, created_at ASC FOR UPDATE`
			args = []interface{}{assetID}
		} else {
			query = `SELECT order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at
					 FROM orders
					 WHERE asset_id = ? AND side = 'SELL' AND status IN ('OPEN', 'PARTIAL') AND price <= ?
					 ORDER BY price ASC, created_at ASC FOR UPDATE`
			args = []interface{}{assetID, price}
		}
	} else {
		if isMarket {
			query = `SELECT order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at
					 FROM orders
					 WHERE asset_id = ? AND side = 'BUY' AND status IN ('OPEN', 'PARTIAL')
					 ORDER BY price DESC, created_at ASC FOR UPDATE`
			args = []interface{}{assetID}
		} else {
			query = `SELECT order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at
					 FROM orders
					 WHERE asset_id = ? AND side = 'BUY' AND status IN ('OPEN', 'PARTIAL') AND price >= ?
					 ORDER BY price DESC, created_at ASC FOR UPDATE`
			args = []interface{}{assetID, price}
		}
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*engine.Order
	for rows.Next() {
		order := &engine.Order{}
		var p sql.NullInt64
		var sp sql.NullInt64
		var filled int64
		var createdAt int64
		var updatedAt int64
		if err := rows.Scan(&order.ID, &order.UserID, &order.AssetID, &order.Side, &order.Type, &order.Quantity, &p, &sp, &filled, &order.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if p.Valid {
			order.Price = p.Int64
		}
		if sp.Valid {
			order.StopPrice = sp.Int64
		}
		order.Remaining = order.Quantity - filled
		order.CreatedAt = time.UnixMilli(createdAt).UTC()
		order.UpdatedAt = time.UnixMilli(updatedAt).UTC()
		orders = append(orders, order)
	}
	return orders, nil
}

func (q *Queries) GetStopOrdersToTrigger(ctx context.Context, tx *sql.Tx, assetID int64, lastPrice int64) ([]*engine.Order, error) {
	query := `SELECT order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at
			  FROM orders
			  WHERE asset_id = ? AND status = 'OPEN' AND (type = 'STOP' OR type = 'STOP_LIMIT')
			  AND ((side = 'BUY' AND stop_price <= ?) OR (side = 'SELL' AND stop_price >= ?))
			  ORDER BY created_at ASC FOR UPDATE`

	rows, err := tx.QueryContext(ctx, query, assetID, lastPrice, lastPrice)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*engine.Order
	for rows.Next() {
		order := &engine.Order{}
		var p sql.NullInt64
		var sp sql.NullInt64
		var filled int64
		var createdAt int64
		var updatedAt int64
		if err := rows.Scan(&order.ID, &order.UserID, &order.AssetID, &order.Side, &order.Type, &order.Quantity, &p, &sp, &filled, &order.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if p.Valid {
			order.Price = p.Int64
		}
		if sp.Valid {
			order.StopPrice = sp.Int64
		}
		order.Remaining = order.Quantity - filled
		order.CreatedAt = time.UnixMilli(createdAt).UTC()
		order.UpdatedAt = time.UnixMilli(updatedAt).UTC()
		orders = append(orders, order)
	}
	return orders, nil
}

func (q *Queries) UpdateOrder(ctx context.Context, order *engine.Order) error {
	filled := order.Quantity - order.Remaining
	_, err := q.Conn.DB.ExecContext(ctx, `
		UPDATE orders SET status = ?, filled_quantity = ?, updated_at = ? WHERE order_id = ?
	`, order.Status, filled, order.UpdatedAt.UnixMilli(), order.ID)
	return err
}

func (q *Queries) UpdateOrderTx(ctx context.Context, tx *sql.Tx, order *engine.Order) error {
	filled := order.Quantity - order.Remaining
	_, err := tx.ExecContext(ctx, `
		UPDATE orders SET status = ?, filled_quantity = ?, updated_at = ?
		WHERE order_id = ?
	`, order.Status, filled, order.UpdatedAt.UnixMilli(), order.ID)
	return err
}

func (q *Queries) InsertExecutionTx(ctx context.Context, tx *sql.Tx, exec engine.Execution, isTakerBuyer bool) error {
	buyOrderID := exec.MakerOrderID
	sellOrderID := exec.TakerOrderID
	if isTakerBuyer {
		buyOrderID = exec.TakerOrderID
		sellOrderID = exec.MakerOrderID
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO executions (buy_order_id, sell_order_id, asset_id, price, quantity, executed_at, is_taker_buyer)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, buyOrderID, sellOrderID, exec.AssetID, exec.Price, exec.Quantity, exec.OccurredAtUTC.UnixMilli(), isTakerBuyer)
	return err
}

func (q *Queries) ListOrders(ctx context.Context) ([]*engine.Order, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at
		FROM orders
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []*engine.Order
	for rows.Next() {
		order := &engine.Order{}
		var price sql.NullInt64
		var stopPrice sql.NullInt64
		var filled int64
		var createdAt int64
		var updatedAt int64
		if err := rows.Scan(&order.ID, &order.UserID, &order.AssetID, &order.Side, &order.Type, &order.Quantity, &price, &stopPrice, &filled, &order.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if price.Valid {
			order.Price = price.Int64
		}
		if stopPrice.Valid {
			order.StopPrice = stopPrice.Int64
		}
		order.Remaining = order.Quantity - filled
		if order.Remaining < 0 {
			order.Remaining = 0
		}
		if createdAt > 0 {
			order.CreatedAt = time.UnixMilli(createdAt).UTC()
		}
		if updatedAt > 0 {
			order.UpdatedAt = time.UnixMilli(updatedAt).UTC()
		} else {
			order.UpdatedAt = order.CreatedAt
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (q *Queries) InsertExecution(ctx context.Context, exec engine.Execution, takerSide engine.Side) error {
	if exec.AssetID == 0 {
		return errors.New("asset id required")
	}
	buyOrderID := exec.MakerOrderID
	sellOrderID := exec.TakerOrderID
	isTakerBuyer := false
	if takerSide == engine.SideBuy {
		buyOrderID = exec.TakerOrderID
		sellOrderID = exec.MakerOrderID
		isTakerBuyer = true
	}
	occurredAt := exec.OccurredAtUTC
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO executions (execution_id, buy_order_id, sell_order_id, asset_id, price, quantity, executed_at, is_taker_buyer)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, exec.ID, buyOrderID, sellOrderID, exec.AssetID, exec.Price, exec.Quantity, occurredAt.UnixMilli(), isTakerBuyer)
	return err
}

func (q *Queries) ListExecutions(ctx context.Context) ([]ExecutionRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT execution_id, buy_order_id, sell_order_id, asset_id, price, quantity, executed_at, is_taker_buyer
		FROM executions
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var executions []ExecutionRecord
	for rows.Next() {
		var record ExecutionRecord
		var executedAt int64
		if err := rows.Scan(&record.ID, &record.BuyOrderID, &record.SellOrderID, &record.AssetID, &record.Price, &record.Quantity, &executedAt, &record.IsTakerBuyer); err != nil {
			return nil, err
		}
		if executedAt > 0 {
			record.ExecutedAt = time.UnixMilli(executedAt).UTC()
		}
		executions = append(executions, record)
	}
	return executions, rows.Err()
}

func (q *Queries) UpsertNewsFeed(ctx context.Context, record NewsRecord) error {
	if record.ID == 0 {
		return errors.New("news id required")
	}
	assetID := interface{}(record.AssetID)
	if record.AssetID == 0 {
		assetID = nil
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO news_feed (news_id, headline, body, published_at, source, sentiment_score, related_asset_id, category, impact, impact_scope)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE headline = VALUES(headline),
			body = VALUES(body),
			published_at = VALUES(published_at),
			source = VALUES(source),
			sentiment_score = VALUES(sentiment_score),
			related_asset_id = VALUES(related_asset_id),
			category = VALUES(category),
			impact = VALUES(impact),
			impact_scope = VALUES(impact_scope)
	`, record.ID, record.Headline, record.Body, record.PublishedAt, record.Source, record.SentimentScore, assetID, record.Category, record.Impact, record.ImpactScope)
	return err
}

func (q *Queries) ListNewsFeed(ctx context.Context) ([]NewsRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT news_id, headline, body, published_at, source, sentiment_score, related_asset_id, category, impact, impact_scope
		FROM news_feed
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NewsRecord
	for rows.Next() {
		var record NewsRecord
		var body sql.NullString
		var source sql.NullString
		var sentiment sql.NullInt64
		var assetID sql.NullInt64
		var category sql.NullString
		var impact sql.NullString
		var impactScope sql.NullString
		if err := rows.Scan(&record.ID, &record.Headline, &body, &record.PublishedAt, &source, &sentiment, &assetID, &category, &impact, &impactScope); err != nil {
			return nil, err
		}
		if body.Valid {
			record.Body = body.String
		}
		if source.Valid {
			record.Source = source.String
		}
		if sentiment.Valid {
			record.SentimentScore = sentiment.Int64
		}
		if assetID.Valid {
			record.AssetID = assetID.Int64
		}
		if category.Valid {
			record.Category = category.String
		}
		if impact.Valid {
			record.Impact = impact.String
		}
		if impactScope.Valid {
			record.ImpactScope = impactScope.String
		}
		items = append(items, record)
	}
	return items, rows.Err()
}

func (q *Queries) GetBalance(ctx context.Context, params models.GetBalanceParams) (int64, error) {
	var amount int64
	err := q.Conn.DB.QueryRowContext(ctx, `
		SELECT amount FROM currency_balances cb
		JOIN currencies c ON cb.currency_id = c.currency_id
		WHERE cb.user_id = ? AND c.code = ?
	`, params.UserID, params.Currency).Scan(&amount)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return amount, err
}

func (q *Queries) GetBalanceRecord(ctx context.Context, params models.GetBalanceParams) (CurrencyBalance, error) {
	var cb CurrencyBalance
	err := q.Conn.DB.QueryRowContext(ctx, `
		SELECT cb.user_id, c.code, cb.amount, cb.locked_amount 
		FROM currency_balances cb
		JOIN currencies c ON cb.currency_id = c.currency_id
		WHERE cb.user_id = ? AND c.code = ?
	`, params.UserID, params.Currency).Scan(&cb.UserID, &cb.Currency, &cb.Amount, &cb.LockedAmount)
	if err == sql.ErrNoRows {
		return CurrencyBalance{UserID: params.UserID, Currency: params.Currency, Amount: 0, LockedAmount: 0}, nil
	}
	return cb, err
}

func (q *Queries) SetCurrencyBalance(ctx context.Context, userID, currencyID, amount int64) error {
	if userID == 0 || currencyID == 0 {
		return errors.New("user id and currency id required")
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO currency_balances (user_id, currency_id, amount, updated_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)
	`, userID, currencyID, amount, time.Now().UTC().UnixMilli())
	if err != nil {
		return err
	}
	return nil
}

func (q *Queries) AdjustCurrencyBalance(ctx context.Context, tx *sql.Tx, userID int64, currency string, delta int64, lockedDelta int64) error {
	var currencyID int64
	var err error
	query := "SELECT currency_id FROM currencies WHERE code = ? LIMIT 1"
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, currency).Scan(&currencyID)
	} else {
		err = q.Conn.DB.QueryRowContext(ctx, query, currency).Scan(&currencyID)
	}
	if err != nil {
		return err
	}

	now := time.Now().UTC().UnixMilli()
	updateQuery := `UPDATE currency_balances SET amount = amount + ?, locked_amount = locked_amount + ?, updated_at = ? WHERE user_id = ? AND currency_id = ?`

	var res sql.Result
	if tx != nil {
		res, err = tx.ExecContext(ctx, updateQuery, delta, lockedDelta, now, userID, currencyID)
	} else {
		res, err = q.Conn.DB.ExecContext(ctx, updateQuery, delta, lockedDelta, now, userID, currencyID)
	}
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		if delta < 0 || lockedDelta < 0 {
			return errors.New("insufficient balance")
		}
		insertQuery := `
			INSERT INTO currency_balances (user_id, currency_id, amount, locked_amount, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`
		if tx != nil {
			_, err = tx.ExecContext(ctx, insertQuery, userID, currencyID, delta, lockedDelta, now)
		} else {
			_, err = q.Conn.DB.ExecContext(ctx, insertQuery, userID, currencyID, delta, lockedDelta, now)
		}
	}
	return err
}

func (q *Queries) AdjustAssetBalance(ctx context.Context, tx *sql.Tx, userID int64, assetID int64, delta int64, lockedDelta int64) error {
	now := time.Now().UTC().UnixMilli()
	updateQuery := `UPDATE asset_balances SET quantity = quantity + ?, locked_quantity = locked_quantity + ?, updated_at = ? WHERE user_id = ? AND asset_id = ?`

	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.ExecContext(ctx, updateQuery, delta, lockedDelta, now, userID, assetID)
	} else {
		res, err = q.Conn.DB.ExecContext(ctx, updateQuery, delta, lockedDelta, now, userID, assetID)
	}
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		if delta < 0 || lockedDelta < 0 {
			return errors.New("insufficient balance")
		}
		insertQuery := `
			INSERT INTO asset_balances (user_id, asset_id, quantity, locked_quantity, average_price, average_acquired_at, updated_at)
			VALUES (?, ?, ?, ?, 0, 0, ?)
		`
		if tx != nil {
			_, err = tx.ExecContext(ctx, insertQuery, userID, assetID, delta, lockedDelta, now)
		} else {
			_, err = q.Conn.DB.ExecContext(ctx, insertQuery, userID, assetID, delta, lockedDelta, now)
		}
	}
	return err
}

func (q *Queries) ListCurrencyBalances(ctx context.Context) ([]CurrencyBalance, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT cb.user_id, c.code, cb.amount, cb.locked_amount
		FROM currency_balances cb
		JOIN currencies c ON c.currency_id = cb.currency_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var balances []CurrencyBalance
	for rows.Next() {
		var balance CurrencyBalance
		if err := rows.Scan(&balance.UserID, &balance.Currency, &balance.Amount, &balance.LockedAmount); err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}
	return balances, rows.Err()
}

func (q *Queries) UpsertAPIKey(ctx context.Context, record APIKeyRecord, createdAt time.Time) error {
	if strings.TrimSpace(record.Key) == "" {
		return errors.New("api key required")
	}
	if record.UserID == 0 {
		return errors.New("user id required")
	}
	if strings.TrimSpace(record.Role) == "" {
		return errors.New("role required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO api_keys (api_key, user_id, role, created_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE user_id = VALUES(user_id), role = VALUES(role)
	`, record.Key, record.UserID, record.Role, createdAt.UnixMilli())
	return err
}

func (q *Queries) ListAPIKeys(ctx context.Context) ([]APIKeyRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, "SELECT api_key, user_id, role FROM api_keys")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []APIKeyRecord
	for rows.Next() {
		var record APIKeyRecord
		if err := rows.Scan(&record.Key, &record.UserID, &record.Role); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (q *Queries) GetPosition(ctx context.Context, params models.GetPositionParams) (int64, error) {
	var qty int64
	err := q.Conn.DB.QueryRowContext(ctx, `
		SELECT quantity FROM asset_balances
		WHERE user_id = ? AND asset_id = ?
	`, params.UserID, params.AssetID).Scan(&qty)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return qty, err
}

func (q *Queries) GetPositionRecord(ctx context.Context, params models.GetPositionParams) (AssetBalance, error) {
	var ab AssetBalance
	err := q.Conn.DB.QueryRowContext(ctx, `
		SELECT user_id, asset_id, quantity, locked_quantity 
		FROM asset_balances 
		WHERE user_id = ? AND asset_id = ?
	`, params.UserID, params.AssetID).Scan(&ab.UserID, &ab.AssetID, &ab.Quantity, &ab.LockedQuantity)
	if err == sql.ErrNoRows {
		return AssetBalance{UserID: params.UserID, AssetID: params.AssetID, Quantity: 0, LockedQuantity: 0}, nil
	}
	return ab, err
}

func (q *Queries) SetAssetBalance(ctx context.Context, userID, assetID, quantity int64) error {
	if userID == 0 || assetID == 0 {
		return errors.New("user id and asset id required")
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO asset_balances (user_id, asset_id, quantity, average_price, average_acquired_at, updated_at)
		VALUES (?, ?, ?, 0, 0, ?)
		ON DUPLICATE KEY UPDATE quantity = VALUES(quantity), updated_at = VALUES(updated_at)
	`, userID, assetID, quantity, time.Now().UTC().UnixMilli())
	return err
}

func (q *Queries) ListAssetBalances(ctx context.Context) ([]AssetBalance, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, "SELECT user_id, asset_id, quantity, locked_quantity FROM asset_balances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var balances []AssetBalance
	for rows.Next() {
		var balance AssetBalance
		if err := rows.Scan(&balance.UserID, &balance.AssetID, &balance.Quantity, &balance.LockedQuantity); err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}
	return balances, rows.Err()
}

func ensureRegion(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	var regionID int64
	err := tx.QueryRowContext(ctx, "SELECT region_id FROM regions WHERE name = ? LIMIT 1", name).Scan(&regionID)
	if err == nil {
		return regionID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO regions (name, description) VALUES (?, '')", name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func ensureCountry(ctx context.Context, tx *sql.Tx, name string, regionID int64) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultCountryName
	}
	var countryID int64
	err := tx.QueryRowContext(ctx, "SELECT country_id FROM countries WHERE name = ? LIMIT 1", name).Scan(&countryID)
	if err == nil {
		return countryID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO countries (region_id, name) VALUES (?, ?)", regionID, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func ensureSector(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultSectorName
	}
	var sectorID int64
	err := tx.QueryRowContext(ctx, "SELECT sector_id FROM sectors WHERE name = ? LIMIT 1", name).Scan(&sectorID)
	if err == nil {
		return sectorID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	code := strings.ToUpper(strings.TrimSpace(name))
	result, err := tx.ExecContext(ctx, "INSERT INTO sectors (code, name) VALUES (?, ?)", code, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (q *Queries) ListCompanies(ctx context.Context) ([]CompanyRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT c.company_id, c.name, c.ticker_symbol,
		       COALESCE(s.name, ''), COALESCE(co.name, ''),
		       c.user_id, c.max_production_capacity, c.current_inventory, c.last_capex_at,
		       c.shares_issued, c.shares_outstanding, c.treasury_stock
		FROM companies c
		LEFT JOIN sectors s ON c.sector_id = s.sector_id
		LEFT JOIN countries co ON c.country_id = co.country_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []CompanyRecord
	for rows.Next() {
		var record CompanyRecord
		if err := rows.Scan(
			&record.CompanyID,
			&record.Name,
			&record.Symbol,
			&record.Sector,
			&record.Country,
			&record.UserID,
			&record.MaxProductionCapacity,
			&record.CurrentInventory,
			&record.LastCapexAt,
			&record.SharesIssued,
			&record.SharesOutstanding,
			&record.TreasuryStock,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertCompany(ctx context.Context, record CompanyRecord) error {
	if record.CompanyID == 0 {
		return errors.New("company id required")
	}
	tx, err := q.Conn.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	regionID, err := ensureRegion(ctx, tx, defaultRegionName)
	if err != nil {
		rollbackTx(tx)
		return err
	}
	countryID, err := ensureCountry(ctx, tx, record.Country, regionID)
	if err != nil {
		rollbackTx(tx)
		return err
	}
	sectorID, err := ensureSector(ctx, tx, record.Sector)
	if err != nil {
		rollbackTx(tx)
		return err
	}
	args := []interface{}{
		record.CompanyID,
		countryID,
		sectorID,
		strings.TrimSpace(record.Name),
		strings.TrimSpace(record.Symbol),
		record.UserID,
		record.MaxProductionCapacity,
		record.CurrentInventory,
		record.LastCapexAt,
		record.SharesIssued,
		record.SharesOutstanding,
		record.TreasuryStock,
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO companies (
			company_id, country_id, sector_id, name, ticker_symbol, description, user_id,
			max_production_capacity, current_inventory, last_capex_at, shares_issued, shares_outstanding, treasury_stock
		) VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			country_id = VALUES(country_id),
			sector_id = VALUES(sector_id),
			name = VALUES(name),
			ticker_symbol = VALUES(ticker_symbol),
			user_id = VALUES(user_id),
			max_production_capacity = VALUES(max_production_capacity),
			current_inventory = VALUES(current_inventory),
			last_capex_at = VALUES(last_capex_at),
			shares_issued = VALUES(shares_issued),
			shares_outstanding = VALUES(shares_outstanding),
			treasury_stock = VALUES(treasury_stock)
	`, args...); err != nil {
		rollbackTx(tx)
		return err
	}
	return tx.Commit()
}

func (q *Queries) ListProductionRecipes(ctx context.Context) ([]ProductionRecipeRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT recipe_id, company_id, output_asset_id, output_quantity
		FROM production_recipes
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []ProductionRecipeRecord
	for rows.Next() {
		var record ProductionRecipeRecord
		if err := rows.Scan(&record.ID, &record.CompanyID, &record.OutputAssetID, &record.OutputQuantity); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListProductionInputs(ctx context.Context) ([]ProductionInputRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT input_id, recipe_id, input_asset_id, input_quantity
		FROM production_inputs
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []ProductionInputRecord
	for rows.Next() {
		var record ProductionInputRecord
		if err := rows.Scan(&record.ID, &record.RecipeID, &record.InputAssetID, &record.InputQuantity); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListFinancialReports(ctx context.Context) ([]FinancialReportRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT company_id, fiscal_year, fiscal_quarter, revenue, net_income, eps, capex, utilization_rate, inventory_level, guidance, published_at
		FROM financial_reports
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []FinancialReportRecord
	for rows.Next() {
		var record FinancialReportRecord
		if err := rows.Scan(
			&record.CompanyID,
			&record.FiscalYear,
			&record.FiscalQuarter,
			&record.Revenue,
			&record.NetIncome,
			&record.EPS,
			&record.Capex,
			&record.UtilizationRate,
			&record.InventoryLevel,
			&record.Guidance,
			&record.PublishedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertFinancialReport(ctx context.Context, report FinancialReportRecord) error {
	if report.CompanyID == 0 {
		return errors.New("company id required")
	}
	args := []interface{}{
		report.CompanyID,
		report.FiscalYear,
		report.FiscalQuarter,
		report.Revenue,
		report.NetIncome,
		report.EPS,
		report.Capex,
		report.UtilizationRate,
		report.InventoryLevel,
		report.Guidance,
		report.PublishedAt,
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO financial_reports (
			company_id, fiscal_year, fiscal_quarter, revenue, net_income, eps, capex, utilization_rate, inventory_level, guidance, published_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			revenue = VALUES(revenue),
			net_income = VALUES(net_income),
			eps = VALUES(eps),
			capex = VALUES(capex),
			utilization_rate = VALUES(utilization_rate),
			inventory_level = VALUES(inventory_level),
			guidance = VALUES(guidance),
			published_at = VALUES(published_at)
	`, args...)
	return err
}

func (q *Queries) ListCompanyDividends(ctx context.Context) ([]CompanyDividendRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT
			company_id, asset_id, fiscal_year, fiscal_quarter, net_income, payout_ratio_bps,
			dividend_per_share, company_payout, pool_payout, spot_payout, margin_long_payout,
			margin_short_charge, eligible_spot_shares, eligible_long_shares, pool_shares, created_at
		FROM company_dividends
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []CompanyDividendRecord
	for rows.Next() {
		var record CompanyDividendRecord
		if err := rows.Scan(
			&record.CompanyID,
			&record.AssetID,
			&record.FiscalYear,
			&record.FiscalQuarter,
			&record.NetIncome,
			&record.PayoutRatioBps,
			&record.DividendPerShare,
			&record.CompanyPayout,
			&record.PoolPayout,
			&record.SpotPayout,
			&record.MarginLongPayout,
			&record.MarginShortCharge,
			&record.EligibleSpotShares,
			&record.EligibleLongShares,
			&record.PoolShares,
			&record.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertCompanyDividend(ctx context.Context, record CompanyDividendRecord) error {
	if record.CompanyID == 0 {
		return errors.New("company id required")
	}
	args := []interface{}{
		record.CompanyID,
		record.AssetID,
		record.FiscalYear,
		record.FiscalQuarter,
		record.NetIncome,
		record.PayoutRatioBps,
		record.DividendPerShare,
		record.CompanyPayout,
		record.PoolPayout,
		record.SpotPayout,
		record.MarginLongPayout,
		record.MarginShortCharge,
		record.EligibleSpotShares,
		record.EligibleLongShares,
		record.PoolShares,
		record.CreatedAt,
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO company_dividends (
			company_id, asset_id, fiscal_year, fiscal_quarter, net_income, payout_ratio_bps, dividend_per_share,
			company_payout, pool_payout, spot_payout, margin_long_payout, margin_short_charge,
			eligible_spot_shares, eligible_long_shares, pool_shares, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			asset_id = VALUES(asset_id),
			net_income = VALUES(net_income),
			payout_ratio_bps = VALUES(payout_ratio_bps),
			dividend_per_share = VALUES(dividend_per_share),
			company_payout = VALUES(company_payout),
			pool_payout = VALUES(pool_payout),
			spot_payout = VALUES(spot_payout),
			margin_long_payout = VALUES(margin_long_payout),
			margin_short_charge = VALUES(margin_short_charge),
			eligible_spot_shares = VALUES(eligible_spot_shares),
			eligible_long_shares = VALUES(eligible_long_shares),
			pool_shares = VALUES(pool_shares),
			created_at = VALUES(created_at)
	`, args...)
	return err
}

func (q *Queries) ListIndexConstituents(ctx context.Context) ([]IndexConstituentRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT index_asset_id, component_asset_id
		FROM index_constituents
		ORDER BY index_asset_id, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []IndexConstituentRecord
	for rows.Next() {
		var r IndexConstituentRecord
		if err := rows.Scan(&r.IndexAssetID, &r.ComponentAssetID); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertIndexConstituents(ctx context.Context, indexAssetID int64, componentAssetIDs []int64) error {
	if indexAssetID == 0 {
		return errors.New("index asset id required")
	}
	createdAt := time.Now().UTC().UnixMilli()
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO assets (asset_id, ticker, type, base_price, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE asset_id = asset_id
	`, indexAssetID, fmt.Sprintf("INDEX-%d", indexAssetID), "INDEX", defaultAssetPrice, createdAt)
	if err != nil {
		return err
	}
	_, err = q.Conn.DB.ExecContext(ctx, `DELETE FROM index_constituents WHERE index_asset_id = ?`, indexAssetID)
	if err != nil {
		return err
	}
	for _, componentID := range componentAssetIDs {
		_, err := q.Conn.DB.ExecContext(ctx, `
			INSERT INTO index_constituents (index_asset_id, component_asset_id)
			VALUES (?, ?)
			ON DUPLICATE KEY UPDATE component_asset_id = VALUES(component_asset_id)
		`, indexAssetID, componentID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *Queries) ListLiquidityPools(ctx context.Context) ([]LiquidityPoolRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT lp.pool_id, c.code, lp.fee_tier_bp, lp.current_tick, lp.liquidity, lp.created_at, lp.tick_spacing, lp.fee_growth_global_0, lp.fee_growth_global_1
		FROM liquidity_pools lp
		JOIN currencies c ON c.currency_id = lp.currency_id
		ORDER BY lp.pool_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]LiquidityPoolRecord, 0)
	for rows.Next() {
		var record LiquidityPoolRecord
		if err := rows.Scan(
			&record.PoolID,
			&record.QuoteCurrency,
			&record.FeeBps,
			&record.CurrentTick,
			&record.Liquidity,
			&record.CreatedAt,
			&record.TickSpacing,
			&record.FeeGrowthBase,
			&record.FeeGrowthQuote,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertLiquidityPool(ctx context.Context, record LiquidityPoolRecord, now time.Time) error {
	if record.PoolID == 0 {
		return errors.New("pool id required")
	}
	quote := strings.TrimSpace(strings.ToUpper(record.QuoteCurrency))
	if quote == "" {
		return errors.New("quote currency required")
	}
	currencyID, err := q.EnsureDefaultCurrency(ctx, quote)
	if err != nil {
		return err
	}
	createdAt := record.CreatedAt
	if createdAt == 0 {
		createdAt = now.UTC().UnixMilli()
	}
	tickSpacing := record.TickSpacing
	if tickSpacing == 0 {
		tickSpacing = 1
	}
	_, err = q.Conn.DB.ExecContext(ctx, `
		INSERT INTO liquidity_pools (
			pool_id, currency_id, fee_tier_bp, current_tick, tick_spacing, liquidity, fee_growth_global_0, fee_growth_global_1, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			currency_id = VALUES(currency_id),
			fee_tier_bp = VALUES(fee_tier_bp),
			current_tick = VALUES(current_tick),
			tick_spacing = VALUES(tick_spacing),
			liquidity = VALUES(liquidity),
			fee_growth_global_0 = VALUES(fee_growth_global_0),
			fee_growth_global_1 = VALUES(fee_growth_global_1)
	`, record.PoolID, currencyID, record.FeeBps, record.CurrentTick, tickSpacing, record.Liquidity, record.FeeGrowthBase, record.FeeGrowthQuote, createdAt)
	return err
}

func (q *Queries) ListLiquidityPositions(ctx context.Context) ([]LiquidityPositionRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT position_id, pool_id, user_id, tick_lower, tick_upper, liquidity, tokens_owed_0, tokens_owed_1, created_at, updated_at
		FROM liquidity_positions
		ORDER BY position_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]LiquidityPositionRecord, 0)
	for rows.Next() {
		var record LiquidityPositionRecord
		if err := rows.Scan(
			&record.ID,
			&record.PoolID,
			&record.UserID,
			&record.LowerTick,
			&record.UpperTick,
			&record.Liquidity,
			&record.BaseAmount,
			&record.QuoteAmount,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertLiquidityPosition(ctx context.Context, record LiquidityPositionRecord, now time.Time) error {
	if record.ID == 0 {
		return errors.New("position id required")
	}
	if record.PoolID == 0 || record.UserID == 0 {
		return errors.New("pool id and user id required")
	}
	createdAt := record.CreatedAt
	if createdAt == 0 {
		createdAt = now.UTC().UnixMilli()
	}
	updatedAt := record.UpdatedAt
	if updatedAt == 0 {
		updatedAt = now.UTC().UnixMilli()
	}
	liquidity := record.Liquidity
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO liquidity_positions (
			position_id, pool_id, user_id, tick_lower, tick_upper, liquidity,
			fee_growth_inside_0_last, fee_growth_inside_1_last, tokens_owed_0, tokens_owed_1,
			is_limit_order, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 0, 0, ?, ?, FALSE, 'ACTIVE', ?, ?)
		ON DUPLICATE KEY UPDATE
			pool_id = VALUES(pool_id),
			user_id = VALUES(user_id),
			tick_lower = VALUES(tick_lower),
			tick_upper = VALUES(tick_upper),
			liquidity = IF(VALUES(liquidity)=0, liquidity, VALUES(liquidity)),
			tokens_owed_0 = VALUES(tokens_owed_0),
			tokens_owed_1 = VALUES(tokens_owed_1),
			updated_at = VALUES(updated_at)
	`, record.ID, record.PoolID, record.UserID, record.LowerTick, record.UpperTick, liquidity, record.BaseAmount, record.QuoteAmount, createdAt, updatedAt)
	return err
}

func (q *Queries) DeleteLiquidityPosition(ctx context.Context, positionID int64) error {
	if positionID == 0 {
		return nil
	}
	_, err := q.Conn.DB.ExecContext(ctx, "DELETE FROM liquidity_positions WHERE position_id = ?", positionID)
	return err
}

func (q *Queries) ListMarginPools(ctx context.Context) ([]MarginPoolRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT mp.pool_id, mp.asset_id, c.code,
		       mp.total_cash, mp.borrowed_cash, mp.total_assets, mp.borrowed_assets,
		       mp.borrow_rate, mp.short_fee, mp.total_cash_shares, mp.total_asset_shares, mp.updated_at
		FROM margin_pools mp
		JOIN currencies c ON c.currency_id = mp.currency_id
		ORDER BY mp.pool_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]MarginPoolRecord, 0)
	for rows.Next() {
		var record MarginPoolRecord
		if err := rows.Scan(
			&record.PoolID,
			&record.AssetID,
			&record.Currency,
			&record.TotalCash,
			&record.BorrowedCash,
			&record.TotalAssets,
			&record.BorrowedAssets,
			&record.CashRateBps,
			&record.AssetRateBps,
			&record.TotalCashShares,
			&record.TotalAssetShares,
			&record.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertMarginPool(ctx context.Context, record MarginPoolRecord, now time.Time) error {
	if record.PoolID == 0 || record.AssetID == 0 {
		return errors.New("pool id and asset id required")
	}
	currencyCode := strings.TrimSpace(strings.ToUpper(record.Currency))
	if currencyCode == "" {
		currencyCode = "ARC"
	}
	currencyID, err := q.EnsureDefaultCurrency(ctx, currencyCode)
	if err != nil {
		return err
	}
	updatedAt := record.UpdatedAt
	if updatedAt == 0 {
		updatedAt = now.UTC().UnixMilli()
	}
	_, err = q.Conn.DB.ExecContext(ctx, `
		INSERT INTO margin_pools (
			pool_id, asset_id, currency_id, total_cash, borrowed_cash, total_assets, borrowed_assets,
			borrow_rate, short_fee, total_cash_shares, total_asset_shares, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			asset_id = VALUES(asset_id),
			currency_id = VALUES(currency_id),
			total_cash = VALUES(total_cash),
			borrowed_cash = VALUES(borrowed_cash),
			total_assets = VALUES(total_assets),
			borrowed_assets = VALUES(borrowed_assets),
			borrow_rate = VALUES(borrow_rate),
			short_fee = VALUES(short_fee),
			total_cash_shares = VALUES(total_cash_shares),
			total_asset_shares = VALUES(total_asset_shares),
			updated_at = VALUES(updated_at)
	`, record.PoolID, record.AssetID, currencyID, record.TotalCash, record.BorrowedCash, record.TotalAssets, record.BorrowedAssets, record.CashRateBps, record.AssetRateBps, record.TotalCashShares, record.TotalAssetShares, updatedAt)
	return err
}

func (q *Queries) ListMarginPoolProviders(ctx context.Context) ([]MarginPoolProviderRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT provider_id, pool_id, user_id, cash_shares, asset_shares, updated_at
		FROM margin_pool_providers
		ORDER BY provider_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]MarginPoolProviderRecord, 0)
	for rows.Next() {
		var record MarginPoolProviderRecord
		if err := rows.Scan(&record.ID, &record.PoolID, &record.UserID, &record.CashShares, &record.AssetShares, &record.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertMarginPoolProvider(ctx context.Context, record MarginPoolProviderRecord, now time.Time) error {
	if record.PoolID == 0 || record.UserID == 0 {
		return errors.New("pool id and user id required")
	}
	updatedAt := record.UpdatedAt
	if updatedAt == 0 {
		updatedAt = now.UTC().UnixMilli()
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO margin_pool_providers (pool_id, user_id, cash_shares, asset_shares, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			cash_shares = VALUES(cash_shares),
			asset_shares = VALUES(asset_shares),
			updated_at = VALUES(updated_at)
	`, record.PoolID, record.UserID, record.CashShares, record.AssetShares, updatedAt)
	return err
}

func (q *Queries) ListMarginPositions(ctx context.Context) ([]MarginPositionRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT position_id, user_id, asset_id, side, quantity, entry_price, current_price, leverage, margin_used, unrealized_pl, created_at, updated_at
		FROM positions
		ORDER BY position_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]MarginPositionRecord, 0)
	for rows.Next() {
		var record MarginPositionRecord
		if err := rows.Scan(
			&record.ID, &record.UserID, &record.AssetID, &record.Side, &record.Quantity,
			&record.EntryPrice, &record.CurrentPrice, &record.Leverage, &record.MarginUsed, &record.UnrealizedPL, &record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertMarginPosition(ctx context.Context, record MarginPositionRecord) error {
	if record.ID == 0 || record.UserID == 0 || record.AssetID == 0 {
		return errors.New("position id, user id and asset id required")
	}
	if strings.TrimSpace(record.Side) == "" {
		return errors.New("position side required")
	}
	side := strings.ToUpper(strings.TrimSpace(record.Side))
	switch side {
	case "BUY", "LONG":
		side = "LONG"
	case "SELL", "SHORT":
		side = "SHORT"
	default:
		return errors.New("invalid position side")
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO positions (
			position_id, user_id, season_id, asset_id, side, quantity, entry_price, current_price, leverage, margin_used, unrealized_pl, created_at, updated_at
		) VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			user_id = VALUES(user_id),
			asset_id = VALUES(asset_id),
			side = VALUES(side),
			quantity = VALUES(quantity),
			entry_price = VALUES(entry_price),
			current_price = VALUES(current_price),
			leverage = VALUES(leverage),
			margin_used = VALUES(margin_used),
			unrealized_pl = VALUES(unrealized_pl),
			updated_at = VALUES(updated_at)
	`, record.ID, record.UserID, record.AssetID, side, record.Quantity, record.EntryPrice, record.CurrentPrice, record.Leverage, record.MarginUsed, record.UnrealizedPL, record.CreatedAt, record.UpdatedAt)
	return err
}

func (q *Queries) DeleteMarginPosition(ctx context.Context, positionID int64) error {
	if positionID == 0 {
		return nil
	}
	_, err := q.Conn.DB.ExecContext(ctx, "DELETE FROM positions WHERE position_id = ?", positionID)
	return err
}

func (q *Queries) ListContracts(ctx context.Context) ([]ContractRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT contract_id, title, COALESCE(description,''), target_asset_id, total_required_quantity, current_delivered_quantity,
		       unit_price, xp_reward_per_unit, min_rank_required, status, start_at, expires_at
		FROM contracts
		ORDER BY contract_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]ContractRecord, 0)
	for rows.Next() {
		var record ContractRecord
		if err := rows.Scan(
			&record.ID, &record.Title, &record.Description, &record.AssetID, &record.TotalRequired, &record.Delivered,
			&record.UnitPrice, &record.XPPerUnit, &record.MinRank, &record.Status, &record.StartAt, &record.ExpiresAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertContract(ctx context.Context, record ContractRecord, now time.Time) error {
	if record.ID == 0 || record.AssetID == 0 {
		return errors.New("contract id and asset id required")
	}
	if strings.TrimSpace(record.Title) == "" {
		return errors.New("contract title required")
	}
	status := strings.TrimSpace(strings.ToUpper(record.Status))
	if status == "" {
		status = "ACTIVE"
	}
	startAt := record.StartAt
	if startAt == 0 {
		startAt = now.UTC().UnixMilli()
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO contracts (
			contract_id, company_id, country_id, title, description, target_asset_id, total_required_quantity, current_delivered_quantity,
			unit_price, xp_reward_per_unit, min_rank_required, status, start_at, expires_at
		) VALUES (?, NULL, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			title = VALUES(title),
			description = VALUES(description),
			target_asset_id = VALUES(target_asset_id),
			total_required_quantity = VALUES(total_required_quantity),
			current_delivered_quantity = VALUES(current_delivered_quantity),
			unit_price = VALUES(unit_price),
			xp_reward_per_unit = VALUES(xp_reward_per_unit),
			min_rank_required = VALUES(min_rank_required),
			status = VALUES(status),
			start_at = VALUES(start_at),
			expires_at = VALUES(expires_at)
	`, record.ID, record.Title, record.Description, record.AssetID, record.TotalRequired, record.Delivered, record.UnitPrice, record.XPPerUnit, record.MinRank, status, startAt, record.ExpiresAt)
	return err
}

func (q *Queries) DeleteContract(ctx context.Context, contractID int64) error {
	if contractID == 0 {
		return nil
	}
	_, err := q.Conn.DB.ExecContext(ctx, "DELETE FROM contracts WHERE contract_id = ?", contractID)
	return err
}

func (q *Queries) InsertContractDelivery(ctx context.Context, record ContractDeliveryRecord, now time.Time) error {
	if record.ContractID == 0 || record.UserID == 0 || record.Quantity <= 0 {
		return errors.New("invalid contract delivery record")
	}
	deliveredAt := record.DeliveredAt
	if deliveredAt == 0 {
		deliveredAt = now.UTC().UnixMilli()
	}
	id := record.ID
	if id == 0 {
		id = deliveredAt
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO contract_deliveries (delivery_id, contract_id, user_id, quantity, payout_amount, xp_gained, delivered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, record.ContractID, record.UserID, record.Quantity, record.PayoutAmount, record.XPGained, deliveredAt)
	return err
}

func (q *Queries) ListContractDeliveries(ctx context.Context) ([]ContractDeliveryRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT delivery_id, contract_id, user_id, quantity, payout_amount, xp_gained, delivered_at
		FROM contract_deliveries
		ORDER BY delivery_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]ContractDeliveryRecord, 0)
	for rows.Next() {
		var record ContractDeliveryRecord
		if err := rows.Scan(&record.ID, &record.ContractID, &record.UserID, &record.Quantity, &record.PayoutAmount, &record.XPGained, &record.DeliveredAt); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListSeasons(ctx context.Context) ([]SeasonRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT season_id, name, COALESCE(theme_code,''), COALESCE(start_at,0), COALESCE(end_at,0), is_active
		FROM seasons
		ORDER BY season_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]SeasonRecord, 0)
	for rows.Next() {
		var record SeasonRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.Theme, &record.StartAt, &record.EndAt, &record.IsActive); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertSeason(ctx context.Context, record SeasonRecord) error {
	if record.ID == 0 || strings.TrimSpace(record.Name) == "" {
		return errors.New("season id and name required")
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO seasons (season_id, name, theme_code, start_at, end_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			theme_code = VALUES(theme_code),
			start_at = VALUES(start_at),
			end_at = VALUES(end_at),
			is_active = VALUES(is_active)
	`, record.ID, record.Name, record.Theme, record.StartAt, record.EndAt, record.IsActive)
	return err
}

func (q *Queries) ListRegions(ctx context.Context) ([]RegionRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT region_id, name, COALESCE(description,'')
		FROM regions
		ORDER BY region_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]RegionRecord, 0)
	for rows.Next() {
		var record RegionRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.Description); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertRegion(ctx context.Context, record RegionRecord) error {
	if record.ID == 0 || strings.TrimSpace(record.Name) == "" {
		return errors.New("region id and name required")
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO regions (region_id, name, description)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			description = VALUES(description)
	`, record.ID, record.Name, record.Description)
	return err
}

func (q *Queries) ListWorldEvents(ctx context.Context) ([]WorldEventRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT event_id, name, COALESCE(description,''), starts_at, ends_at
		FROM world_events
		ORDER BY event_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]WorldEventRecord, 0)
	for rows.Next() {
		var record WorldEventRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.Description, &record.StartsAt, &record.EndsAt); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) UpsertWorldEvent(ctx context.Context, record WorldEventRecord) error {
	if record.ID == 0 || strings.TrimSpace(record.Name) == "" {
		return errors.New("event id and name required")
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO world_events (event_id, name, description, starts_at, ends_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			description = VALUES(description),
			starts_at = VALUES(starts_at),
			ends_at = VALUES(ends_at)
	`, record.ID, record.Name, record.Description, record.StartsAt, record.EndsAt)
	return err
}

func (q *Queries) ListMacroIndicators(ctx context.Context) ([]MacroIndicatorRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT co.name, mi.type, mi.value, mi.published_at
		FROM macro_indicators mi
		JOIN countries co ON co.country_id = mi.country_id
		ORDER BY mi.indicator_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]MacroIndicatorRecord, 0)
	for rows.Next() {
		var record MacroIndicatorRecord
		if err := rows.Scan(&record.Country, &record.Type, &record.Value, &record.PublishedAt); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ReplaceMacroIndicators(ctx context.Context, records []MacroIndicatorRecord) error {
	tx, err := q.Conn.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM macro_indicators"); err != nil {
		rollbackTx(tx)
		return err
	}
	regionID, err := ensureRegion(ctx, tx, defaultRegionName)
	if err != nil {
		rollbackTx(tx)
		return err
	}
	for _, record := range records {
		country := strings.TrimSpace(record.Country)
		kind := strings.TrimSpace(record.Type)
		if country == "" || kind == "" {
			continue
		}
		switch kind {
		case "GDP_GROWTH", "CPI", "INTEREST_RATE", "UNEMPLOYMENT", "CONSUMER_CONFIDENCE":
		default:
			continue
		}
		countryID, err := ensureCountry(ctx, tx, country, regionID)
		if err != nil {
			rollbackTx(tx)
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO macro_indicators (country_id, type, value, published_at)
			VALUES (?, ?, ?, ?)
		`, countryID, kind, record.Value, record.PublishedAt); err != nil {
			rollbackTx(tx)
			return err
		}
	}
	return tx.Commit()
}

// Server State Management
func (q *Queries) GetServerStateBool(ctx context.Context, key string, defaultValue bool) (bool, error) {
	var value bool
	err := q.Conn.DB.QueryRowContext(ctx, "SELECT state_value FROM server_state WHERE state_key = ?", key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultValue, nil
		}
		return defaultValue, err
	}
	return value, nil
}

func (q *Queries) SetServerStateBool(ctx context.Context, key string, value bool) error {
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO server_state (state_key, state_value)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE state_value = VALUES(state_value)
	`, key, value)
	return err
}

type TransactionLogRecord struct {
	UserID       int64
	CurrencyID   int64
	Amount       int64
	BalanceAfter int64
	Type         string
	ReferenceID  string
	Description  string
	CreatedAt    int64
}

func (q *Queries) InsertTransactionLog(ctx context.Context, tx *sql.Tx, record TransactionLogRecord) error {
	query := `
		INSERT INTO transaction_logs (user_id, currency_id, amount, balance_after, type, reference_id, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, record.UserID, record.CurrencyID, record.Amount, record.BalanceAfter, record.Type, record.ReferenceID, record.Description, record.CreatedAt)
	} else {
		_, err = q.Conn.DB.ExecContext(ctx, query, record.UserID, record.CurrencyID, record.Amount, record.BalanceAfter, record.Type, record.ReferenceID, record.Description, record.CreatedAt)
	}
	return err
}

type MarketCandleRecord struct {
	AssetID   int64
	Timeframe string
	OpenTime  int64
	Open      int64
	High      int64
	Low       int64
	Close     int64
	Volume    int64
}

func (q *Queries) UpsertMarketCandle(ctx context.Context, record MarketCandleRecord) error {
	query := `
		INSERT INTO market_candles (asset_id, timeframe, open_time, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			high = GREATEST(high, VALUES(high)),
			low = LEAST(low, VALUES(low)),
			close = VALUES(close),
			volume = volume + VALUES(volume)
	`
	_, err := q.Conn.DB.ExecContext(ctx, query, record.AssetID, record.Timeframe, record.OpenTime, record.Open, record.High, record.Low, record.Close, record.Volume)
	return err
}

type CountryRecord struct {
	ID       int64
	RegionID int64
	Name     string
}

type CurrencyRecord struct {
	ID        int64
	CountryID int64
	Code      string
	Name      string
}

type SectorRecord struct {
	ID   int64
	Code string
	Name string
}

type RankDefinitionRecord struct {
	ID                  int
	Name                string
	RequiredXP          int64
	MakerFeeBps10       int64
	TakerFeeBps10       int64
	InterestDiscountBps int64
	FXFeeDiscountBps    int64
}

type ResourceRecord struct {
	ID          int64
	Name        string
	Type        string
	Description string
}

func (q *Queries) InsertResource(ctx context.Context, record ResourceRecord) (int64, error) {
	query := "INSERT INTO resources (name, type, description) VALUES (?, ?, ?)"
	res, err := q.Conn.DB.ExecContext(ctx, query, record.Name, record.Type, record.Description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) ListResources(ctx context.Context) ([]ResourceRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `SELECT resource_id, name, type, COALESCE(description,'') FROM resources`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []ResourceRecord
	for rows.Next() {
		var record ResourceRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.Type, &record.Description); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListCountries(ctx context.Context) ([]CountryRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `SELECT country_id, region_id, name FROM countries`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []CountryRecord
	for rows.Next() {
		var record CountryRecord
		if err := rows.Scan(&record.ID, &record.RegionID, &record.Name); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListCurrencies(ctx context.Context) ([]CurrencyRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `SELECT currency_id, COALESCE(country_id,0), code, name FROM currencies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []CurrencyRecord
	for rows.Next() {
		var record CurrencyRecord
		if err := rows.Scan(&record.ID, &record.CountryID, &record.Code, &record.Name); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListSectors(ctx context.Context) ([]SectorRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `SELECT sector_id, code, name FROM sectors`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []SectorRecord
	for rows.Next() {
		var record SectorRecord
		if err := rows.Scan(&record.ID, &record.Code, &record.Name); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListRankDefinitions(ctx context.Context) ([]RankDefinitionRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT rank_id, name, required_xp, maker_fee_bps10, taker_fee_bps10, interest_discount_bps, fx_fee_discount_bps
		FROM rank_definitions ORDER BY required_xp ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []RankDefinitionRecord
	for rows.Next() {
		var record RankDefinitionRecord
		if err := rows.Scan(
			&record.ID, &record.Name, &record.RequiredXP, &record.MakerFeeBps10, &record.TakerFeeBps10,
			&record.InterestDiscountBps, &record.FXFeeDiscountBps,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListMarketCandles(ctx context.Context, assetID int64) ([]MarketCandleRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT asset_id, timeframe, open_time, open, high, low, close, volume
		FROM market_candles WHERE asset_id = ? ORDER BY open_time ASC
	`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []MarketCandleRecord
	for rows.Next() {
		var record MarketCandleRecord
		if err := rows.Scan(
			&record.AssetID, &record.Timeframe, &record.OpenTime, &record.Open,
			&record.High, &record.Low, &record.Close, &record.Volume,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) ListTransactionLogs(ctx context.Context, userID int64) ([]TransactionLogRecord, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT user_id, currency_id, amount, balance_after, type, COALESCE(reference_id,''), COALESCE(description,''), created_at
		FROM transaction_logs WHERE user_id = ? ORDER BY log_id ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []TransactionLogRecord
	for rows.Next() {
		var record TransactionLogRecord
		if err := rows.Scan(
			&record.UserID, &record.CurrencyID, &record.Amount, &record.BalanceAfter,
			&record.Type, &record.ReferenceID, &record.Description, &record.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (q *Queries) InsertProductionRecipe(ctx context.Context, record ProductionRecipeRecord) (int64, error) {
	query := "INSERT INTO production_recipes (company_id, output_asset_id, output_quantity) VALUES (?, ?, ?)"
	res, err := q.Conn.DB.ExecContext(ctx, query, record.CompanyID, record.OutputAssetID, record.OutputQuantity)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (q *Queries) InsertProductionInput(ctx context.Context, record ProductionInputRecord) (int64, error) {
	query := "INSERT INTO production_inputs (recipe_id, input_asset_id, input_quantity) VALUES (?, ?, ?)"
	res, err := q.Conn.DB.ExecContext(ctx, query, record.RecipeID, record.InputAssetID, record.InputQuantity)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func rollbackTx(tx *sql.Tx) {
	if tx == nil {
		return
	}
	_ = tx.Rollback()
}
