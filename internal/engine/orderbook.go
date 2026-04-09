package engine

import (
	"context"
	"errors"
	"time"
)

const defaultFindOrderTimeout = 2 * time.Second

var ErrOrderRejected = errors.New("order rejected")
var ErrOrderNotFound = errors.New("order not found")

type OrderResult struct {
	Order      *Order      `json:"order"`
	Executions []Execution `json:"executions"`
	Err        error       `json:"-"`
}

type OrderBookSnapshot struct {
	AssetID   int64   `json:"asset_id"`
	LastPrice int64   `json:"last_price"`
	Bids      []Level `json:"bids"`
	Asks      []Level `json:"asks"`
}

type Level struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
}

type orderRequestType int

const (
	requestSubmit orderRequestType = iota
	requestCancel
	requestSnapshot
	requestFind
)

type orderRequest struct {
	action   orderRequestType
	ctx      context.Context
	order    *Order
	orderID  int64
	depth    int
	response chan OrderResult
	snapshot chan OrderBookSnapshot
	find     chan *Order
}

type OrderBook struct {
	assetID  int64
	requests chan orderRequest
	shutdown chan struct{}
	done     chan struct{}
	storage  Storage
}

func NewOrderBook(assetID int64, events EventSink, storage Storage) *OrderBook {
	// eventSink is now handled inside storage implementation or as callbacks.
	return &OrderBook{
		assetID:  assetID,
		requests: make(chan orderRequest, 128),
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
		storage:  storage,
	}
}

func (ob *OrderBook) Start() {
	go ob.run()
}

func (ob *OrderBook) Stop() {
	select {
	case <-ob.shutdown:
		return
	default:
		close(ob.shutdown)
	}
	<-ob.done
}

func (ob *OrderBook) Submit(ctx context.Context, order *Order) (OrderResult, error) {
	response := make(chan OrderResult, 1)
	request := orderRequest{
		action:   requestSubmit,
		ctx:      ctx,
		order:    order,
		response: response,
	}
	if err := ob.enqueue(ctx, request); err != nil {
		return OrderResult{}, err
	}
	return ob.waitResult(ctx, response)
}

func (ob *OrderBook) Cancel(ctx context.Context, orderID int64) (OrderResult, error) {
	response := make(chan OrderResult, 1)
	request := orderRequest{
		action:   requestCancel,
		ctx:      ctx,
		orderID:  orderID,
		response: response,
	}
	if err := ob.enqueue(ctx, request); err != nil {
		return OrderResult{}, err
	}
	return ob.waitResult(ctx, response)
}

func (ob *OrderBook) Snapshot(ctx context.Context, depth int) (OrderBookSnapshot, error) {
	snapshot := make(chan OrderBookSnapshot, 1)
	request := orderRequest{
		action:   requestSnapshot,
		ctx:      ctx,
		depth:    depth,
		snapshot: snapshot,
	}
	if err := ob.enqueue(ctx, request); err != nil {
		return OrderBookSnapshot{}, err
	}
	select {
	case result := <-snapshot:
		return result, nil
	case <-ctx.Done():
		return OrderBookSnapshot{}, ctx.Err()
	}
}

func (ob *OrderBook) FindOrder(orderID int64) (*Order, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFindOrderTimeout)
	defer cancel()
	response := make(chan *Order, 1)
	request := orderRequest{
		action:  requestFind,
		ctx:     ctx,
		orderID: orderID,
		find:    response,
	}
	if err := ob.enqueue(ctx, request); err != nil {
		return nil, false
	}
	var order *Order
	select {
	case order = <-response:
	case <-ctx.Done():
		return nil, false
	}
	if order == nil {
		return nil, false
	}
	return order, true
}

func (ob *OrderBook) enqueue(ctx context.Context, request orderRequest) error {
	select {
	case ob.requests <- request:
		return nil
	case <-ob.shutdown:
		return errors.New("orderbook stopped")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (ob *OrderBook) waitResult(ctx context.Context, response <-chan OrderResult) (OrderResult, error) {
	select {
	case result := <-response:
		return result, result.Err
	case <-ctx.Done():
		return OrderResult{}, ctx.Err()
	}
}

func (ob *OrderBook) run() {
	defer close(ob.done)
	for {
		select {
		case request := <-ob.requests:
			switch request.action {
			case requestSubmit:
				res, _ := ob.storage.ProcessSubmit(request.ctx, request.order)
				request.response <- res
			case requestCancel:
				res, _ := ob.storage.ProcessCancel(request.ctx, request.orderID)
				request.response <- res
			case requestSnapshot:
				snap, _ := ob.storage.GetOrderBookSnapshot(request.ctx, ob.assetID, request.depth)
				request.snapshot <- snap
			case requestFind:
				order, _ := ob.storage.FindOrder(request.ctx, request.orderID)
				request.find <- order
			}
		case <-ob.shutdown:
			return
		}
	}
}
