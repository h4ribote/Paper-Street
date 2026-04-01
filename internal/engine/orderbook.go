package engine

import (
	"context"
	"errors"
	"sort"
	"time"
)

const defaultGuardPercent int64 = 5
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
	order    *Order
	orderID  int64
	depth    int
	response chan OrderResult
	snapshot chan OrderBookSnapshot
	find     chan *Order
}

type OrderBook struct {
	assetID      int64
	bids         *priceLevels
	asks         *priceLevels
	stopOrders   []*Order
	lastPrice    int64
	requests     chan orderRequest
	shutdown     chan struct{}
	done         chan struct{}
	nextOrderID  int64
	orders       map[int64]*Order
	guardPercent int64
	events       EventSink
}

func NewOrderBook(assetID int64, events EventSink) *OrderBook {
	if events == nil {
		events = NewDiscardSink()
	}
	return &OrderBook{
		assetID:      assetID,
		bids:         newPriceLevels(SideBuy),
		asks:         newPriceLevels(SideSell),
		requests:     make(chan orderRequest, 128),
		shutdown:     make(chan struct{}),
		done:         make(chan struct{}),
		orders:       make(map[int64]*Order),
		guardPercent: defaultGuardPercent,
		events:       events,
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
				request.response <- ob.handleSubmit(request.order)
			case requestCancel:
				request.response <- ob.handleCancel(request.orderID)
			case requestSnapshot:
				request.snapshot <- ob.snapshot(request.depth)
			case requestFind:
				request.find <- ob.findOrder(request.orderID)
			}
		case <-ob.shutdown:
			return
		}
	}
}

func (ob *OrderBook) handleSubmit(order *Order) OrderResult {
	if order == nil {
		return OrderResult{Err: errors.New("order required")}
	}
	if order.Quantity <= 0 {
		order.reject()
		return OrderResult{Order: order.clone(), Err: ErrOrderRejected}
	}
	if order.ID == 0 {
		ob.nextOrderID++
		order.ID = ob.nextOrderID
	}
	order.AssetID = ob.assetID
	order.CreatedAt = time.Now().UTC()
	order.Remaining = order.Quantity
	order.open()
	ob.orders[order.ID] = order

	if order.isStopOrder() {
		ob.stopOrders = append(ob.stopOrders, order)
		ob.events.EnqueueOrder(order)
		return OrderResult{Order: order.clone()}
	}

	result := ob.match(order)
	ob.events.EnqueueOrder(order)
	for _, exec := range result.Executions {
		ob.events.EnqueueExecution(exec)
	}
	return result
}

func (ob *OrderBook) handleCancel(orderID int64) OrderResult {
	order, ok := ob.orders[orderID]
	if !ok {
		return OrderResult{Err: ErrOrderNotFound}
	}
	if !order.isActive() {
		return OrderResult{Order: order.clone()}
	}
	if order.isStopOrder() {
		ob.removeStopOrder(order.ID)
		order.cancel()
		ob.events.EnqueueOrder(order)
		return OrderResult{Order: order.clone()}
	}
	removed := false
	if order.Side == SideBuy {
		removed = ob.bids.remove(order)
	} else {
		removed = ob.asks.remove(order)
	}
	if !removed {
		return OrderResult{Order: order.clone()}
	}
	order.cancel()
	ob.events.EnqueueOrder(order)
	return OrderResult{Order: order.clone()}
}

func (ob *OrderBook) findOrder(orderID int64) *Order {
	order, ok := ob.orders[orderID]
	if !ok {
		return nil
	}
	return order.clone()
}

func (ob *OrderBook) snapshot(depth int) OrderBookSnapshot {
	if depth <= 0 {
		depth = 20
	}
	return OrderBookSnapshot{
		AssetID:   ob.assetID,
		LastPrice: ob.lastPrice,
		Bids:      ob.bids.snapshot(depth, true),
		Asks:      ob.asks.snapshot(depth, false),
	}
}

func (ob *OrderBook) match(order *Order) OrderResult {
	result := OrderResult{Order: order}
	var executions []Execution
	var guardPrice int64
	guardEnabled := order.Type == OrderTypeMarket
	timeInForce := order.effectiveTimeInForce()
	if timeInForce == TimeInForceFOK && !ob.canFill(order) {
		order.cancel()
		return OrderResult{Order: order.clone(), Executions: nil}
	}
	selfTradeReduced := false
	for order.Remaining > 0 {
		maker := ob.bestOpposing(order.Side)
		if maker == nil {
			break
		}
		if !ob.isCrossing(order, maker.Price) {
			break
		}
		if maker.UserID == order.UserID {
			selfTradeQty := minInt64(order.Remaining, maker.Remaining)
			ob.removeMaker(maker)
			maker.cancel()
			ob.events.EnqueueOrder(maker)
			if order.Type == OrderTypeMarket && timeInForce != TimeInForceFOK {
				order.Remaining -= selfTradeQty
				selfTradeReduced = true
				if order.Remaining == 0 {
					break
				}
			}
			continue
		}
		if guardEnabled && guardPrice == 0 {
			guardPrice = ob.guardFrom(maker.Price, order.Side)
		}
		if guardEnabled && !ob.guardSatisfied(maker.Price, guardPrice, order.Side) {
			break
		}
		fillQty := minInt64(order.Remaining, maker.Remaining)
		exec := newExecution(ob.assetID, maker.Price, fillQty, order, maker)
		executions = append(executions, exec)
		maker.fill(fillQty)
		order.fill(fillQty)
		ob.lastPrice = maker.Price
		ob.events.EnqueueOrder(maker)
		if maker.Remaining == 0 {
			ob.removeMaker(maker)
		}
		if order.Remaining == 0 {
			break
		}
	}
	result.Executions = executions
	ob.triggerStopOrders()

	if selfTradeReduced && order.Remaining == 0 {
		if len(executions) == 0 {
			order.cancel()
		} else {
			order.Status = OrderStatusPartial
		}
		return OrderResult{Order: order.clone(), Executions: executions}
	}

	if order.Remaining == 0 {
		order.Status = OrderStatusFilled
		return OrderResult{Order: order.clone(), Executions: executions}
	}
	if order.Type == OrderTypeMarket {
		if len(executions) == 0 {
			order.cancel()
			return OrderResult{Order: order.clone(), Executions: executions}
		}
		order.Status = OrderStatusPartial
		order.Remaining = 0
		return OrderResult{Order: order.clone(), Executions: executions}
	}

	if timeInForce == TimeInForceIOC {
		if len(executions) == 0 {
			order.cancel()
			return OrderResult{Order: order.clone(), Executions: executions}
		}
		order.Status = OrderStatusPartial
		order.Remaining = 0
		return OrderResult{Order: order.clone(), Executions: executions}
	}

	if timeInForce == TimeInForceFOK {
		order.cancel()
		order.Remaining = 0
		return OrderResult{Order: order.clone(), Executions: executions}
	}

	if order.isStopOrder() {
		return OrderResult{Order: order.clone(), Executions: executions}
	}

	if len(executions) > 0 {
		order.Status = OrderStatusPartial
	} else {
		order.Status = OrderStatusOpen
	}
	ob.addToBook(order)
	return OrderResult{Order: order.clone(), Executions: executions}
}

func (ob *OrderBook) canFill(order *Order) bool {
	if order == nil {
		return false
	}
	if order.Remaining <= 0 {
		return true
	}
	guardEnabled := order.Type == OrderTypeMarket
	var guardPrice int64
	if guardEnabled {
		maker := ob.bestOpposing(order.Side)
		if maker == nil {
			return false
		}
		guardPrice = ob.guardFrom(maker.Price, order.Side)
		if guardPrice == 0 {
			return false
		}
	}
	remaining := order.Remaining
	if order.Side == SideBuy {
		for i := 0; i < len(ob.asks.prices) && remaining > 0; i++ {
			price := ob.asks.prices[i]
			if !ob.priceAcceptable(order, price, guardEnabled, guardPrice) {
				break
			}
			for _, maker := range ob.asks.levels[price] {
				if maker.UserID == order.UserID {
					continue
				}
				remaining -= maker.Remaining
				if remaining <= 0 {
					return true
				}
			}
		}
		return false
	}
	for i := len(ob.bids.prices) - 1; i >= 0 && remaining > 0; i-- {
		price := ob.bids.prices[i]
		if !ob.priceAcceptable(order, price, guardEnabled, guardPrice) {
			break
		}
		for _, maker := range ob.bids.levels[price] {
			if maker.UserID == order.UserID {
				continue
			}
			remaining -= maker.Remaining
			if remaining <= 0 {
				return true
			}
		}
	}
	return false
}

func (ob *OrderBook) priceAcceptable(order *Order, price int64, guardEnabled bool, guardPrice int64) bool {
	if guardEnabled && !ob.guardSatisfied(price, guardPrice, order.Side) {
		return false
	}
	switch order.Type {
	case OrderTypeLimit, OrderTypeStopLimit:
		if order.Side == SideBuy {
			return price <= order.Price
		}
		return price >= order.Price
	default:
		return true
	}
}

func (ob *OrderBook) addToBook(order *Order) {
	if order.Side == SideBuy {
		ob.bids.add(order)
	} else {
		ob.asks.add(order)
	}
}

func (ob *OrderBook) bestOpposing(side Side) *Order {
	if side == SideBuy {
		return ob.asks.best()
	}
	return ob.bids.best()
}

func (ob *OrderBook) isCrossing(order *Order, opposingPrice int64) bool {
	switch order.Type {
	case OrderTypeMarket:
		return true
	case OrderTypeLimit, OrderTypeStopLimit:
		if order.Side == SideBuy {
			return opposingPrice <= order.Price
		}
		return opposingPrice >= order.Price
	default:
		return false
	}
}

func (ob *OrderBook) guardFrom(bestPrice int64, side Side) int64 {
	if bestPrice <= 0 {
		return 0
	}
	if side == SideBuy {
		return bestPrice * (100 + ob.guardPercent) / 100
	}
	return bestPrice * (100 - ob.guardPercent) / 100
}

func (ob *OrderBook) guardSatisfied(price int64, guard int64, side Side) bool {
	if guard == 0 {
		return true
	}
	if side == SideBuy {
		return price <= guard
	}
	return price >= guard
}

func (ob *OrderBook) removeMaker(order *Order) {
	if order.Side == SideBuy {
		ob.bids.remove(order)
	} else {
		ob.asks.remove(order)
	}
}

func (ob *OrderBook) triggerStopOrders() {
	if len(ob.stopOrders) == 0 {
		return
	}
	triggered := make([]*Order, 0)
	remaining := make([]*Order, 0)
	for _, stop := range ob.stopOrders {
		if !stop.isActive() {
			continue
		}
		if ob.shouldTrigger(stop) {
			triggered = append(triggered, stop)
		} else {
			remaining = append(remaining, stop)
		}
	}
	ob.stopOrders = remaining
	for _, stop := range triggered {
		ob.activateStopOrder(stop)
	}
}

func (ob *OrderBook) shouldTrigger(order *Order) bool {
	if ob.lastPrice == 0 {
		return false
	}
	if order.Side == SideBuy {
		return ob.lastPrice >= order.StopPrice
	}
	return ob.lastPrice <= order.StopPrice
}

func (ob *OrderBook) activateStopOrder(order *Order) {
	if order.Type == OrderTypeStop {
		order.Type = OrderTypeMarket
	} else {
		order.Type = OrderTypeLimit
	}
	result := ob.match(order)
	ob.events.EnqueueOrder(order)
	for _, exec := range result.Executions {
		ob.events.EnqueueExecution(exec)
	}
}

func (ob *OrderBook) removeStopOrder(orderID int64) {
	filtered := make([]*Order, 0, len(ob.stopOrders))
	for _, order := range ob.stopOrders {
		if order.ID != orderID {
			filtered = append(filtered, order)
		}
	}
	ob.stopOrders = filtered
}

func newPriceLevels(side Side) *priceLevels {
	return &priceLevels{
		side:   side,
		levels: make(map[int64][]*Order),
	}
}

type priceLevels struct {
	side   Side
	prices []int64
	levels map[int64][]*Order
}

func (pl *priceLevels) add(order *Order) {
	if order == nil {
		return
	}
	queue, ok := pl.levels[order.Price]
	if ok {
		pl.levels[order.Price] = append(queue, order)
		return
	}
	pl.levels[order.Price] = []*Order{order}
	pl.insertPrice(order.Price)
}

func (pl *priceLevels) best() *Order {
	if len(pl.prices) == 0 {
		return nil
	}
	price := pl.bestPrice()
	queue := pl.levels[price]
	if len(queue) == 0 {
		return nil
	}
	return queue[0]
}

func (pl *priceLevels) remove(order *Order) bool {
	if order == nil {
		return false
	}
	queue, ok := pl.levels[order.Price]
	if !ok {
		return false
	}
	for i, current := range queue {
		if current.ID == order.ID {
			queue = append(queue[:i], queue[i+1:]...)
			if len(queue) == 0 {
				delete(pl.levels, order.Price)
				pl.removePrice(order.Price)
			} else {
				pl.levels[order.Price] = queue
			}
			return true
		}
	}
	return false
}

func (pl *priceLevels) snapshot(depth int, descending bool) []Level {
	if depth <= 0 {
		return nil
	}
	levels := make([]Level, 0, depth)
	prices := pl.prices
	if descending {
		for i := len(prices) - 1; i >= 0 && len(levels) < depth; i-- {
			price := prices[i]
			levels = append(levels, Level{Price: price, Quantity: sumRemaining(pl.levels[price])})
		}
		return levels
	}
	for i := 0; i < len(prices) && len(levels) < depth; i++ {
		price := prices[i]
		levels = append(levels, Level{Price: price, Quantity: sumRemaining(pl.levels[price])})
	}
	return levels
}

func (pl *priceLevels) bestPrice() int64 {
	if pl.side == SideBuy {
		return pl.prices[len(pl.prices)-1]
	}
	return pl.prices[0]
}

func (pl *priceLevels) insertPrice(price int64) {
	index := sort.Search(len(pl.prices), func(i int) bool { return pl.prices[i] >= price })
	pl.prices = append(pl.prices, 0)
	copy(pl.prices[index+1:], pl.prices[index:])
	pl.prices[index] = price
}

func (pl *priceLevels) removePrice(price int64) {
	index := sort.Search(len(pl.prices), func(i int) bool { return pl.prices[i] >= price })
	if index >= len(pl.prices) || pl.prices[index] != price {
		return
	}
	pl.prices = append(pl.prices[:index], pl.prices[index+1:]...)
}

func sumRemaining(orders []*Order) int64 {
	var total int64
	for _, order := range orders {
		total += order.Remaining
	}
	return total
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
