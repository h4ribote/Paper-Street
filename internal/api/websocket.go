package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gws "github.com/gorilla/websocket"

	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

const (
	wsCloseNormal           = 4000
	wsCloseInvalidToken     = 4001
	wsCloseInvalidMessage   = 4002
	wsCloseTooManySubs      = 4003
	wsCloseInternal         = 5000
	wsMaxSubscriptions      = 100
	wsDefaultBroadcastEvery = time.Second
	wsSnapshotTimeout       = 500 * time.Millisecond
	wsSnapshotDepth         = 20
	wsTradeLimit            = 50
	wsNewsLimit             = 20
	wsCandleLimit           = 60
	wsMaxMessageSize        = 1 << 20 // 1MB
)

var (
	errTopicsRequired    = errors.New("topics required")
	errSubscriptionLimit = errors.New("subscription limit exceeded")
)

type wsRequest struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type wsMessage struct {
	Topic string      `json:"topic"`
	Data  interface{} `json:"data"`
	TS    int64       `json:"ts"`
}

type wsPortfolioSnapshot struct {
	Balances    []models.Balance   `json:"balances"`
	Positions   []models.Position  `json:"positions"`
	Assets      []PortfolioAsset   `json:"assets"`
	Performance []PerformancePoint `json:"performance"`
}

type wsClient struct {
	id             string
	userID         int64
	conn           *gws.Conn
	send           chan wsMessage
	subscriptions  map[string]struct{}
	orderbookSnaps map[string]engine.OrderBookSnapshot
	mu             sync.RWMutex
	closed         bool
}

func (c *wsClient) subscribe(topics []string) ([]string, error) {
	if len(topics) == 0 {
		return nil, errTopicsRequired
	}
	added := make([]string, 0, len(topics))
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		if _, ok := c.subscriptions[topic]; ok {
			continue
		}
		if len(c.subscriptions) >= wsMaxSubscriptions {
			return nil, errSubscriptionLimit
		}
		c.subscriptions[topic] = struct{}{}
		added = append(added, topic)
	}
	return added, nil
}

func (c *wsClient) unsubscribe(topics []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, topic := range topics {
		trimmed := strings.TrimSpace(topic)
		delete(c.subscriptions, trimmed)
		if strings.HasPrefix(trimmed, "market.orderbook.") {
			delete(c.orderbookSnaps, trimmed)
		}
	}
}

func (c *wsClient) subscriptionsSnapshot() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	topics := make([]string, 0, len(c.subscriptions))
	for topic := range c.subscriptions {
		topics = append(topics, topic)
	}
	return topics
}

func (c *wsClient) enqueue(message wsMessage) bool {
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()
	if closed {
		return false
	}
	select {
	case c.send <- message:
		return true
	default:
		return false
	}
}

func (c *wsClient) close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	close(c.send)
	c.mu.Unlock()
	_ = c.conn.Close()
}

func (c *wsClient) setOrderbookSnapshot(topic string, snapshot engine.OrderBookSnapshot) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.orderbookSnaps == nil {
		c.orderbookSnaps = make(map[string]engine.OrderBookSnapshot)
	}
	c.orderbookSnaps[topic] = snapshot
	c.mu.Unlock()
}

func (c *wsClient) orderbookSnapshot(topic string) (engine.OrderBookSnapshot, bool) {
	if c == nil {
		return engine.OrderBookSnapshot{}, false
	}
	c.mu.RLock()
	snapshot, ok := c.orderbookSnaps[topic]
	c.mu.RUnlock()
	return snapshot, ok
}

type wsHub struct {
	store     *MarketStore
	engine    *engine.Engine
	mu        sync.RWMutex
	clients   map[string]*wsClient
	nextID    uint64
	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
}

func newWSHub(store *MarketStore, engine *engine.Engine) *wsHub {
	return &wsHub{
		store:   store,
		engine:  engine,
		clients: make(map[string]*wsClient),
		stopCh:  make(chan struct{}),
	}
}

func (h *wsHub) Start(interval time.Duration) {
	h.startOnce.Do(func() {
		if interval <= 0 {
			interval = wsDefaultBroadcastEvery
		}
		ticker := time.NewTicker(interval)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					h.broadcastSnapshots()
				case <-h.stopCh:
					return
				}
			}
		}()
	})
}

func (h *wsHub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
}

func (h *wsHub) register(client *wsClient) {
	h.mu.Lock()
	h.clients[client.id] = client
	h.mu.Unlock()
}

func (h *wsHub) unregister(client *wsClient) {
	h.mu.Lock()
	delete(h.clients, client.id)
	h.mu.Unlock()
	client.close()
}

func (h *wsHub) clientsSnapshot() []*wsClient {
	h.mu.RLock()
	clients := make([]*wsClient, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	return clients
}

func (h *wsHub) broadcastSnapshots() {
	if h.store == nil {
		return
	}
	clients := h.clientsSnapshot()
	for _, client := range clients {
		topics := client.subscriptionsSnapshot()
		for _, topic := range topics {
			h.sendSnapshotOrDelta(client, topic)
		}
	}
}

func (h *wsHub) sendSnapshotOrDelta(client *wsClient, topic string) {
	if strings.HasPrefix(topic, "market.orderbook.") {
		h.sendOrderbookDelta(client, topic)
		return
	}
	h.sendSnapshot(client, topic)
}

func (h *wsHub) sendSnapshot(client *wsClient, topic string) {
	if client == nil {
		return
	}
	data, ok := h.snapshotForTopic(client, topic)
	if !ok {
		return
	}
	if snapshot, isSnapshot := data.(engine.OrderBookSnapshot); isSnapshot {
		client.setOrderbookSnapshot(topic, snapshot)
	}
	client.enqueue(wsMessage{
		Topic: topic,
		Data:  data,
		TS:    time.Now().UTC().UnixMilli(),
	})
}

func (h *wsHub) sendOrderbookDelta(client *wsClient, topic string) {
	if client == nil {
		return
	}
	data, ok := h.snapshotForTopic(client, topic)
	if !ok {
		return
	}
	snapshot, ok := data.(engine.OrderBookSnapshot)
	if !ok {
		return
	}
	previous, ok := client.orderbookSnapshot(topic)
	if !ok {
		client.setOrderbookSnapshot(topic, snapshot)
		client.enqueue(wsMessage{
			Topic: topic,
			Data:  snapshot,
			TS:    time.Now().UTC().UnixMilli(),
		})
		return
	}
	delta, changed := orderbookDelta(previous, snapshot)
	if !changed {
		return
	}
	client.setOrderbookSnapshot(topic, snapshot)
	client.enqueue(wsMessage{
		Topic: topic,
		Data:  delta,
		TS:    time.Now().UTC().UnixMilli(),
	})
}

func (h *wsHub) snapshotForTopic(client *wsClient, topic string) (interface{}, bool) {
	if h.store == nil {
		return nil, false
	}
	switch {
	case topic == "market.ticker":
		return h.store.Tickers(), true
	case topic == "news":
		return h.store.News(wsNewsLimit), true
	case topic == "user.orders":
		if client.userID == 0 {
			return nil, false
		}
		return h.store.Orders(OrderFilter{UserID: client.userID}), true
	case topic == "user.executions":
		if client.userID == 0 {
			return nil, false
		}
		return h.store.TradeHistory(client.userID, wsTradeLimit), true
	case topic == "user.portfolio":
		if client.userID == 0 {
			return nil, false
		}
		return wsPortfolioSnapshot{
			Balances:    h.store.Balances(client.userID),
			Positions:   h.store.Positions(client.userID),
			Assets:      h.store.PortfolioAssets(client.userID),
			Performance: h.store.Performance(client.userID),
		}, true
	case strings.HasPrefix(topic, "market.orderbook."):
		assetID, ok := parseTopicAssetID(topic, "market.orderbook.")
		if !ok || h.engine == nil {
			return nil, false
		}
		ctx, cancel := context.WithTimeout(context.Background(), wsSnapshotTimeout)
		defer cancel()
		// Initialize the order book goroutine so snapshots succeed even before any orders arrive.
		h.engine.OrderBook(assetID)
		snapshot, err := h.engine.Snapshot(ctx, assetID, wsSnapshotDepth)
		if err != nil {
			return nil, false
		}
		return snapshot, true
	case strings.HasPrefix(topic, "market.trade."):
		assetID, ok := parseTopicAssetID(topic, "market.trade.")
		if !ok {
			return nil, false
		}
		return h.store.Executions(assetID, wsTradeLimit), true
	case strings.HasPrefix(topic, "market.candles."):
		assetID, timeframe, ok := parseCandleTopic(topic)
		if !ok {
			return nil, false
		}
		return h.store.Candles(assetID, timeframe, time.Time{}, time.Time{}, wsCandleLimit), true
	default:
		return nil, false
	}
}

func orderbookDelta(previous, current engine.OrderBookSnapshot) (engine.OrderBookSnapshot, bool) {
	bids, bidsChanged := diffLevels(previous.Bids, current.Bids, true)
	asks, asksChanged := diffLevels(previous.Asks, current.Asks, false)
	lastPriceChanged := previous.LastPrice != current.LastPrice
	if !bidsChanged {
		bids = []engine.Level{}
	}
	if !asksChanged {
		asks = []engine.Level{}
	}
	if !bidsChanged && !asksChanged && !lastPriceChanged {
		return engine.OrderBookSnapshot{}, false
	}
	return engine.OrderBookSnapshot{
		AssetID:   current.AssetID,
		LastPrice: current.LastPrice,
		Bids:      bids,
		Asks:      asks,
	}, true
}

func diffLevels(previous, current []engine.Level, sortDescending bool) ([]engine.Level, bool) {
	if len(previous) == 0 && len(current) == 0 {
		return nil, false
	}
	previousMap := make(map[int64]int64, len(previous))
	for _, level := range previous {
		previousMap[level.Price] = level.Quantity
	}
	currentMap := make(map[int64]int64, len(current))
	for _, level := range current {
		currentMap[level.Price] = level.Quantity
	}
	changed := make([]engine.Level, 0, len(current))
	for price, qty := range currentMap {
		if prevQty, ok := previousMap[price]; !ok || prevQty != qty {
			changed = append(changed, engine.Level{Price: price, Quantity: qty})
		}
	}
	for price := range previousMap {
		if _, ok := currentMap[price]; !ok {
			changed = append(changed, engine.Level{Price: price, Quantity: 0})
		}
	}
	if len(changed) == 0 {
		return nil, false
	}
	sort.Slice(changed, func(i, j int) bool {
		if sortDescending {
			return changed[i].Price > changed[j].Price
		}
		return changed[i].Price < changed[j].Price
	})
	return changed, true
}

func parseTopicAssetID(topic, prefix string) (int64, bool) {
	trimmed := strings.TrimPrefix(topic, prefix)
	if trimmed == "" {
		return 0, false
	}
	assetID, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || assetID <= 0 {
		return 0, false
	}
	return assetID, true
}

func parseCandleTopic(topic string) (int64, time.Duration, bool) {
	segments := strings.Split(topic, ".")
	if len(segments) < 4 {
		return 0, 0, false
	}
	assetID, err := strconv.ParseInt(segments[2], 10, 64)
	if err != nil || assetID <= 0 {
		return 0, 0, false
	}
	timeframe, ok := parseTimeframe(segments[3])
	if !ok {
		return 0, 0, false
	}
	return assetID, timeframe, true
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.WSHub == nil {
		respondError(w, http.StatusInternalServerError, "websocket unavailable")
		return
	}
	if s.APIKeys == nil {
		respondError(w, http.StatusInternalServerError, "auth unavailable")
		return
	}
	s.WSHub.Start(wsDefaultBroadcastEvery)
	apiKey := strings.TrimSpace(r.URL.Query().Get("api_key"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(r.Header.Get(apiKeyHeader))
	}
	upgrader := gws.Upgrader{
		CheckOrigin: wsCheckOrigin,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	if apiKey == "" || !s.APIKeys.ContainsHex(apiKey) {
		writeWSClose(conn, wsCloseInvalidToken, "invalid api key")
		return
	}
	var userID int64
	if s.Store != nil {
		if user, ok := s.Store.UserForAPIKey(apiKey); ok {
			userID = user.ID
		}
	}
	clientID := strconv.FormatUint(atomic.AddUint64(&s.WSHub.nextID, 1), 10)
	client := &wsClient{
		id:             clientID,
		userID:         userID,
		conn:           conn,
		send:           make(chan wsMessage, 256),
		subscriptions:  make(map[string]struct{}),
		orderbookSnaps: make(map[string]engine.OrderBookSnapshot),
	}
	s.WSHub.register(client)
	go s.wsWriteLoop(client)
	s.wsReadLoop(client)
}

func (s *Server) wsReadLoop(client *wsClient) {
	defer s.WSHub.unregister(client)
	client.conn.SetReadLimit(wsMaxMessageSize)
	for {
		var req wsRequest
		if err := client.conn.ReadJSON(&req); err != nil {
			var closeErr *gws.CloseError
			if errors.As(err, &closeErr) {
				return
			}
			writeWSClose(client.conn, wsCloseInvalidMessage, "invalid message")
			return
		}
		op := strings.ToLower(strings.TrimSpace(req.Op))
		switch op {
		case "subscribe":
			topics, err := client.subscribe(req.Args)
			if err != nil {
				if errors.Is(err, errTopicsRequired) {
					writeWSClose(client.conn, wsCloseInvalidMessage, err.Error())
				} else {
					writeWSClose(client.conn, wsCloseTooManySubs, err.Error())
				}
				return
			}
			for _, topic := range topics {
				s.WSHub.sendSnapshot(client, topic)
			}
		case "unsubscribe":
			client.unsubscribe(req.Args)
		case "ping":
			client.enqueue(wsMessage{Topic: "pong", Data: "ok", TS: time.Now().UTC().UnixMilli()})
		default:
			writeWSClose(client.conn, wsCloseInvalidMessage, "invalid operation")
			return
		}
	}
}

func (s *Server) wsWriteLoop(client *wsClient) {
	for message := range client.send {
		if err := client.conn.WriteJSON(message); err != nil {
			writeWSClose(client.conn, wsCloseInternal, "write failed")
			return
		}
	}
}

func writeWSClose(conn *gws.Conn, code int, reason string) {
	_ = conn.WriteControl(gws.CloseMessage, gws.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	_ = conn.Close()
}

func wsCheckOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(originURL.Host, r.Host)
}
