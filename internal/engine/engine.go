package engine

import (
	"context"
	"errors"
	"sync"
)

var ErrOrderBookNotFound = errors.New("orderbook not found")

// EventSink persists order and execution updates asynchronously.
type EventSink interface {
	EnqueueOrder(order *Order)
	EnqueueExecution(execution Execution)
	Shutdown(ctx context.Context) error
}

type Engine struct {
	mu         sync.RWMutex
	orderBooks map[int64]*OrderBook
	sink       EventSink
}

func NewEngine(sink EventSink) *Engine {
	if sink == nil {
		sink = NewDiscardSink()
	}
	return &Engine{
		orderBooks: make(map[int64]*OrderBook),
		sink:       sink,
	}
}

func (e *Engine) OrderBook(assetID int64) *OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()
	book := e.orderBooks[assetID]
	if book != nil {
		return book
	}
	book = NewOrderBook(assetID, e.sink)
	e.orderBooks[assetID] = book
	book.Start()
	return book
}

func (e *Engine) SubmitOrder(ctx context.Context, order *Order) (OrderResult, error) {
	book := e.OrderBook(order.AssetID)
	return book.Submit(ctx, order)
}

func (e *Engine) CancelOrder(ctx context.Context, assetID int64, orderID int64) (OrderResult, error) {
	book, ok := e.getOrderBook(assetID)
	if !ok {
		return OrderResult{}, ErrOrderBookNotFound
	}
	return book.Cancel(ctx, orderID)
}

func (e *Engine) CancelOrderByID(ctx context.Context, orderID int64) (OrderResult, error) {
	e.mu.RLock()
	books := make([]*OrderBook, 0, len(e.orderBooks))
	for _, book := range e.orderBooks {
		books = append(books, book)
	}
	e.mu.RUnlock()
	for _, book := range books {
		result, err := book.Cancel(ctx, orderID)
		if err == ErrOrderNotFound {
			continue
		}
		return result, err
	}
	return OrderResult{}, ErrOrderNotFound
}

func (e *Engine) Snapshot(ctx context.Context, assetID int64, depth int) (OrderBookSnapshot, error) {
	book, ok := e.getOrderBook(assetID)
	if !ok {
		return OrderBookSnapshot{}, ErrOrderBookNotFound
	}
	return book.Snapshot(ctx, depth)
}

// FindOrder looks up an order within the specified asset's order book.
// It returns false if the asset ID is incorrect, the asset has no order book,
// or the order isn't found in that book.
func (e *Engine) FindOrder(assetID int64, orderID int64) (*Order, bool) {
	book, ok := e.getOrderBook(assetID)
	if !ok {
		return nil, false
	}
	return book.FindOrder(orderID)
}

func (e *Engine) Shutdown(ctx context.Context) error {
	e.mu.RLock()
	books := make([]*OrderBook, 0, len(e.orderBooks))
	for _, book := range e.orderBooks {
		books = append(books, book)
	}
	e.mu.RUnlock()
	for _, book := range books {
		book.Stop()
	}
	return e.sink.Shutdown(ctx)
}

func (e *Engine) getOrderBook(assetID int64) (*OrderBook, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	book, ok := e.orderBooks[assetID]
	return book, ok
}
