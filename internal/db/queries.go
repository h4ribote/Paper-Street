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
	defaultCurrencyName = "Arcadian Credit"
	defaultUserRank     = "Shrimp"
)

type Queries struct {
	Conn *Connection
}

type AssetSnapshot struct {
	Asset     models.Asset
	BasePrice int64
}

type CurrencyBalance struct {
	UserID   int64
	Currency string
	Amount   int64
}

type AssetBalance struct {
	UserID   int64
	AssetID  int64
	Quantity int64
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
	rank := strings.TrimSpace(user.Rank)
	if rank == "" {
		rank = defaultUserRank
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO users (user_id, username, rank, created_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE username = VALUES(username), rank = VALUES(rank)
	`, user.ID, strings.TrimSpace(user.Username), rank, createdAt.UnixMilli())
	return err
}

func (q *Queries) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, "SELECT user_id, username, rank FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Rank); err != nil {
			return nil, err
		}
		user.Role = "player"
		users = append(users, user)
	}
	return users, rows.Err()
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
	// Fallback for ticker so inserts don't violate NOT NULL/UNIQUE constraints.
	if asset.Symbol == "" {
		asset.Symbol = fmt.Sprintf("ASSET-%d", asset.ID)
	}
	if asset.Type == "" {
		asset.Type = "STOCK"
	}
	createdAt := time.Now().UTC().UnixMilli()
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO assets (asset_id, ticker, type, base_price, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE ticker = VALUES(ticker), type = VALUES(type), base_price = VALUES(base_price)
	`, asset.ID, strings.TrimSpace(asset.Symbol), strings.TrimSpace(asset.Type), basePrice, createdAt)
	return err
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
		if asset.Sector == "" {
			asset.Sector = "GENERAL"
		}
		assets = append(assets, AssetSnapshot{Asset: asset, BasePrice: basePrice})
	}
	return assets, rows.Err()
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
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO orders (order_id, user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

func (q *Queries) SetCurrencyBalance(ctx context.Context, userID, currencyID, amount int64) error {
	if userID == 0 || currencyID == 0 {
		return errors.New("user id and currency id required")
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO currency_balances (user_id, currency_id, amount, updated_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)
	`, userID, currencyID, amount, time.Now().UTC().UnixMilli())
	return err
}

func (q *Queries) ListCurrencyBalances(ctx context.Context) ([]CurrencyBalance, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, `
		SELECT cb.user_id, c.code, cb.amount
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
		if err := rows.Scan(&balance.UserID, &balance.Currency, &balance.Amount); err != nil {
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
	rows, err := q.Conn.DB.QueryContext(ctx, "SELECT user_id, asset_id, quantity FROM asset_balances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var balances []AssetBalance
	for rows.Next() {
		var balance AssetBalance
		if err := rows.Scan(&balance.UserID, &balance.AssetID, &balance.Quantity); err != nil {
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
	args := []interface{}{
		record.CompanyID,
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
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO companies (
			company_id, country_id, sector_id, name, ticker_symbol, description, user_id,
			max_production_capacity, current_inventory, last_capex_at, shares_issued, shares_outstanding, treasury_stock
		) VALUES (?, NULL, NULL, ?, ?, '', ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			ticker_symbol = VALUES(ticker_symbol),
			user_id = VALUES(user_id),
			max_production_capacity = VALUES(max_production_capacity),
			current_inventory = VALUES(current_inventory),
			last_capex_at = VALUES(last_capex_at),
			shares_issued = VALUES(shares_issued),
			shares_outstanding = VALUES(shares_outstanding),
			treasury_stock = VALUES(treasury_stock)
	`, args...)
	return err
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
		ORDER BY index_asset_id, component_asset_id
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
	_, err := q.Conn.DB.ExecContext(ctx, `DELETE FROM index_constituents WHERE index_asset_id = ?`, indexAssetID)
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
