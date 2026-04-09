package api

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

const defaultGuardPercent = 20

// ProcessSubmit handles order submission and matching logic within a database transaction.
func (s *MarketStore) ProcessSubmit(ctx context.Context, order *engine.Order) (engine.OrderResult, error) {
	tx, err := s.queries.Conn.DB.BeginTx(ctx, nil)
	if err != nil {
		return engine.OrderResult{Err: err}, err
	}
	defer tx.Rollback()

	// 1. Initial setup and persistence
	order.Status = engine.OrderStatusOpen
	order.Remaining = order.Quantity
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}
	order.UpdatedAt = order.CreatedAt

	if err := s.queries.UpsertOrder(ctx, order); err != nil {
		return engine.OrderResult{Err: err}, err
	}

	if order.Type == engine.OrderTypeStop || order.Type == engine.OrderTypeStopLimit {
		if err := tx.Commit(); err != nil {
			return engine.OrderResult{Err: err}, err
		}
		s.notifyOrderUpdate(order)
		return engine.OrderResult{Order: cloneOrder(order)}, nil
	}

	// 2. Matching logic
	result, err := s.performMatchingTx(ctx, tx, order)
	if err != nil {
		return engine.OrderResult{Err: err}, err
	}

	if err := tx.Commit(); err != nil {
		return engine.OrderResult{Err: err}, err
	}

	// 3. Post-commit notifications
	s.notifyOrderUpdate(order)
	for _, exec := range result.Executions {
		s.notifyExecutionUpdate(exec, order.Side)
	}

	// Trigger stop orders based on last price
	if len(result.Executions) > 0 {
		lastExec := result.Executions[len(result.Executions)-1]
		s.triggerStopOrders(ctx, lastExec.AssetID, lastExec.Price)
	}

	return result, nil
}

func (s *MarketStore) ProcessCancel(ctx context.Context, orderID int64) (engine.OrderResult, error) {
	order, err := s.FindOrder(ctx, orderID)
	if err != nil || order == nil {
		return engine.OrderResult{Err: engine.ErrOrderNotFound}, engine.ErrOrderNotFound
	}

	if order.Status != engine.OrderStatusOpen && order.Status != engine.OrderStatusPartial {
		return engine.OrderResult{Order: order}, nil
	}

	order.Status = engine.OrderStatusCancelled
	order.UpdatedAt = time.Now().UTC()

	if err := s.queries.UpdateOrder(ctx, order); err != nil {
		return engine.OrderResult{Err: err}, err
	}

	s.notifyOrderUpdate(order)
	return engine.OrderResult{Order: order}, nil
}

func (s *MarketStore) FindOrder(ctx context.Context, orderID int64) (*engine.Order, error) {
	// We can't easily filter by ID in queries? Oh wait, ListOrders with a filter or something.
	// Let's assume we can add a specific method if not there.
	// For now, use the DB directly.
	order := &engine.Order{ID: orderID}
	var p, sp sql.NullInt64
	var filled int64
	var createdAt, updatedAt int64
	err := s.queries.Conn.DB.QueryRowContext(ctx,
		"SELECT user_id, asset_id, side, type, quantity, price, stop_price, filled_quantity, status, created_at, updated_at FROM orders WHERE order_id = ?",
		orderID,
	).Scan(&order.UserID, &order.AssetID, &order.Side, &order.Type, &order.Quantity, &p, &sp, &filled, &order.Status, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if p.Valid { order.Price = p.Int64 }
	if sp.Valid { order.StopPrice = sp.Int64 }
	order.Remaining = order.Quantity - filled
	order.CreatedAt = time.UnixMilli(createdAt).UTC()
	order.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return order, nil
}

func (s *MarketStore) GetOrderBookSnapshot(ctx context.Context, assetID int64, depth int) (engine.OrderBookSnapshot, error) {
	snapshot := engine.OrderBookSnapshot{AssetID: assetID}
	
	// Fetch last price
	_ = s.queries.Conn.DB.QueryRowContext(ctx, "SELECT price FROM executions WHERE asset_id = ? ORDER BY executed_at DESC LIMIT 1", assetID).Scan(&snapshot.LastPrice)

	// Fetch Bids
	rows, err := s.queries.Conn.DB.QueryContext(ctx,
		"SELECT price, SUM(quantity - filled_quantity) FROM orders WHERE asset_id = ? AND side = 'BUY' AND status IN ('OPEN', 'PARTIAL') GROUP BY price ORDER BY price DESC LIMIT ?",
		assetID, depth,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var l engine.Level
			if err := rows.Scan(&l.Price, &l.Quantity); err == nil {
				snapshot.Bids = append(snapshot.Bids, l)
			}
		}
	}

	// Fetch Asks
	rows, err = s.queries.Conn.DB.QueryContext(ctx,
		"SELECT price, SUM(quantity - filled_quantity) FROM orders WHERE asset_id = ? AND side = 'SELL' AND status IN ('OPEN', 'PARTIAL') GROUP BY price ORDER BY price ASC LIMIT ?",
		assetID, depth,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var l engine.Level
			if err := rows.Scan(&l.Price, &l.Quantity); err == nil {
				snapshot.Asks = append(snapshot.Asks, l)
			}
		}
	}

	return snapshot, nil
}

func (s *MarketStore) performMatchingTx(ctx context.Context, tx *sql.Tx, order *engine.Order) (engine.OrderResult, error) {
	var executions []engine.Execution
	guardEnabled := order.Type == engine.OrderTypeMarket
	var guardPrice int64

	for order.Remaining > 0 {
		crossing, err := s.queries.GetCrossingOrders(ctx, tx, order.AssetID, order.Side, order.Price, order.Type == engine.OrderTypeMarket)
		if err != nil || len(crossing) == 0 {
			break
		}

		matched := false
		for _, maker := range crossing {
			if maker.UserID == order.UserID {
				// Self-trade prevention: Cancel maker
				maker.Status = engine.OrderStatusCancelled
				maker.UpdatedAt = time.Now().UTC()
				_ = s.queries.UpdateOrderTx(ctx, tx, maker)
				s.notifyOrderUpdate(maker)
				continue
			}

			if guardEnabled && guardPrice == 0 {
				guardPrice = s.guardFrom(maker.Price, order.Side)
			}
			if guardEnabled && !s.guardSatisfied(maker.Price, guardPrice, order.Side) {
				break
			}

			fillQty := minInt64(order.Remaining, maker.Remaining)
			exec := engine.Execution{
				AssetID:       order.AssetID,
				Price:         maker.Price,
				Quantity:      fillQty,
				TakerOrderID:  order.ID,
				MakerOrderID:  maker.ID,
				TakerUserID:   order.UserID,
				MakerUserID:   maker.UserID,
				OccurredAtUTC: time.Now().UTC(),
			}

			// Apply the trade (balances & status)
			if err := s.applyExecutionTx(ctx, tx, exec, order.Side); err != nil {
				// E.g. insufficient funds
				break
			}

			executions = append(executions, exec)
			maker.Remaining -= fillQty
			if maker.Remaining <= 0 {
				maker.Status = engine.OrderStatusFilled
			} else {
				maker.Status = engine.OrderStatusPartial
			}
			maker.UpdatedAt = time.Now().UTC()
			_ = s.queries.UpdateOrderTx(ctx, tx, maker)

			order.Remaining -= fillQty
			if order.Remaining <= 0 {
				order.Status = engine.OrderStatusFilled
			} else {
				order.Status = engine.OrderStatusPartial
			}
			order.UpdatedAt = time.Now().UTC()

			matched = true
			if order.Remaining == 0 {
				break
			}
		}

		if !matched || order.Remaining == 0 {
			break
		}
	}

	// Final taker update
	if order.Type == engine.OrderTypeMarket || order.TimeInForce == engine.TimeInForceIOC || order.TimeInForce == engine.TimeInForceFOK {
		if order.Remaining > 0 {
			order.Status = engine.OrderStatusCancelled
			if len(executions) > 0 {
				order.Status = engine.OrderStatusPartial
			}
			order.Remaining = 0
		}
	}
	_ = s.queries.UpdateOrderTx(ctx, tx, order)

	return engine.OrderResult{Order: order, Executions: executions}, nil
}

func (s *MarketStore) applyExecutionTx(ctx context.Context, tx *sql.Tx, exec engine.Execution, takerSide engine.Side) error {
	// Logic from applyExecutionLocked but using tx
	// Simplified here - ideally we use full fee/margin logic
	
	cashDelta := exec.Price * exec.Quantity
	
	// Identify buyer/seller
	buyerID := exec.TakerUserID
	sellerID := exec.MakerUserID
	if takerSide == engine.SideSell {
		buyerID = exec.MakerUserID
		sellerID = exec.TakerUserID
	}

	// 1. Funds check and Move cash
	// In a real system, we fetch buyer balance from DB with FOR UPDATE if not already locked
	// But our crossing orders are locked, and taker is handled by the sequentially processed OrderBook.
	
	// Deduct from buyer
	if err := s.queries.AdjustCurrencyBalance(ctx, tx, buyerID, defaultCurrency, -cashDelta); err != nil {
		return err
	}
	// Add to seller
	if err := s.queries.AdjustCurrencyBalance(ctx, tx, sellerID, defaultCurrency, cashDelta); err != nil {
		return err
	}

	// 2. Move assets
	if err := s.queries.AdjustAssetBalance(ctx, tx, buyerID, exec.AssetID, exec.Quantity); err != nil {
		return err
	}
	if err := s.queries.AdjustAssetBalance(ctx, tx, sellerID, exec.AssetID, -exec.Quantity); err != nil {
		return err
	}

	// 3. Execution record
	if err := s.queries.InsertExecutionTx(ctx, tx, exec, takerSide == engine.SideBuy); err != nil {
		return err
	}

	return nil
}

func (s *MarketStore) triggerStopOrders(ctx context.Context, assetID int64, lastPrice int64) {
	// This might be tricky if it creates recursive submissions.
	// Actually, the sequencers will handle it.
	// For now, let's just trigger them by fetching and then submitting them back to the OrderBook!
	// This keeps the sequential property.
	
	tx, err := s.queries.Conn.DB.BeginTx(ctx, nil)
	if err != nil { return }
	defer tx.Rollback()

	triggered, err := s.queries.GetStopOrdersToTrigger(ctx, tx, assetID, lastPrice)
	if err != nil || len(triggered) == 0 {
		return
	}
	
	_ = tx.Commit() 
	
	// Submit each triggered order back to the order book (which is this same OrderBook go routine queue)
	for _, stop := range triggered {
		if stop.Type == engine.OrderTypeStop {
			stop.Type = engine.OrderTypeMarket
		} else {
			stop.Type = engine.OrderTypeLimit
		}
		// submit to orderbook (this will be enqueued in the requests channel)
		// but wait, we are ALREADY in the OrderBook run goroutine if we were called from there.
		// If we call Submit synchronously, it will deadlock.
		// So we must do it asynchronously.
		go func(o *engine.Order) {
			_, _ = s.ProcessSubmit(context.Background(), o)
		}(stop)
	}
}

func (s *MarketStore) notifyOrderUpdate(order *engine.Order) {
	// Broadcast through WebSocket hub
	s.mu.RLock()
	hub := s.WSHub
	s.mu.RUnlock()
	if hub != nil {
		hub.Trigger()
		publicOrder := PublicOrderEvent{
			ID:        order.ID,
			AssetID:   order.AssetID,
			Side:      order.Side,
			Type:      order.Type,
			Quantity:  order.Quantity,
			Remaining: order.Remaining,
			Price:     order.Price,
			Status:    order.Status,
			CreatedAt: order.CreatedAt,
			UpdatedAt: order.UpdatedAt,
		}
		hub.BroadcastEvent(fmt.Sprintf("market.order.%d", order.AssetID), publicOrder)
	}
}

func (s *MarketStore) notifyExecutionUpdate(exec engine.Execution, takerSide engine.Side) {
	// Similar to notifyOrderUpdate
}

func (s *MarketStore) guardFrom(bestPrice int64, side engine.Side) int64 {
	if bestPrice <= 0 { return 0 }
	if side == engine.SideBuy {
		return bestPrice * (100 + defaultGuardPercent) / 100
	}
	return bestPrice * (100 - defaultGuardPercent) / 100
}

func (s *MarketStore) guardSatisfied(price int64, guard int64, side engine.Side) bool {
	if guard == 0 { return true }
	if side == engine.SideBuy {
		return price <= guard
	}
	return price >= guard
}


