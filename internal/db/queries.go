package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	defaultRegionName   = "Global"
	defaultCountryName  = "United States"
	defaultCurrencyName = "US Dollar"
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
		return q.EnsureDefaultCurrency(ctx, code)
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

func (q *Queries) UpsertUser(ctx context.Context, user models.User, createdAt time.Time) error {
	if user.ID == 0 {
		return errors.New("user id required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := q.Conn.DB.ExecContext(ctx, `
		INSERT INTO users (user_id, username, rank, created_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE username = VALUES(username), rank = VALUES(rank)
	`, user.ID, strings.TrimSpace(user.Username), defaultUserRank, createdAt.UnixMilli())
	return err
}

func (q *Queries) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := q.Conn.DB.QueryContext(ctx, "SELECT user_id, username FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username); err != nil {
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
		asset.Symbol = strings.ToUpper(strings.TrimSpace(asset.Name))
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
	assets := []AssetSnapshot{}
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
	orders := []*engine.Order{}
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
	executions := []ExecutionRecord{}
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
	balances := []CurrencyBalance{}
	for rows.Next() {
		var balance CurrencyBalance
		if err := rows.Scan(&balance.UserID, &balance.Currency, &balance.Amount); err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}
	return balances, rows.Err()
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
	balances := []AssetBalance{}
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
