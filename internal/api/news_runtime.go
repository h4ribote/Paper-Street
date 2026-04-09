package api

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

const (
	defaultNewsInterval      = 45 * time.Second
	defaultNewsBaseQuantity  = int64(40)
	defaultNewsMinConfidence = 0.1
	defaultNewsImpactFactor  = 0.05
	defaultNewsImpactJitter  = 0.15
	newsOrderTimeout         = 2 * time.Second
	newsReactorUserID        = int64(900001)
	newsLiquidityUserID      = int64(900002)
	newsCashBufferMultiplier = int64(2)
)

type NewsEngineConfig struct {
	Interval      time.Duration
	BaseQuantity  int64
	MinConfidence float64
	ImpactFactor  float64
	ImpactJitter  float64
}

func DefaultNewsEngineConfig() NewsEngineConfig {
	return NewsEngineConfig{
		Interval:      defaultNewsInterval,
		BaseQuantity:  defaultNewsBaseQuantity,
		MinConfidence: defaultNewsMinConfidence,
		ImpactFactor:  defaultNewsImpactFactor,
		ImpactJitter:  defaultNewsImpactJitter,
	}
}

func (c NewsEngineConfig) withDefaults() NewsEngineConfig {
	if c.Interval <= 0 {
		c.Interval = defaultNewsInterval
	}
	if c.BaseQuantity <= 0 {
		c.BaseQuantity = defaultNewsBaseQuantity
	}
	if c.MinConfidence <= 0 {
		c.MinConfidence = defaultNewsMinConfidence
	}
	if c.ImpactFactor <= 0 {
		c.ImpactFactor = defaultNewsImpactFactor
	}
	if c.ImpactJitter < 0 {
		c.ImpactJitter = defaultNewsImpactJitter
	}
	return c
}

func StartNewsEngine(ctx context.Context, store *MarketStore, eng *engine.Engine, cfg NewsEngineConfig) {
	if store == nil || eng == nil {
		return
	}
	cfg = cfg.withDefaults()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ticker := time.NewTicker(cfg.Interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				store.generateNewsTick(time.Now().UTC(), eng, rng, cfg)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *MarketStore) generateNewsTick(now time.Time, eng *engine.Engine, rng *rand.Rand, cfg NewsEngineConfig) (NewsItem, bool) {
	item, ok := s.randomNewsItem(now, rng)
	if !ok {
		return NewsItem{}, false
	}
	item = s.publishNewsItem(now, item)
	s.applyNewsImpact(item, eng, rng, cfg)
	return item, true
}

func (s *MarketStore) publishNewsItemLocked(now time.Time, item NewsItem) NewsItem {
	if item.PublishedAt == 0 {
		item.PublishedAt = now.UnixMilli()
	}
	// We don't have a news slice anymore, it's in the DB.
	// But we might want a News() method to retrieve it.
	// run asynchronously so it doesn't try to use the DB connection on the same thread
	// if we are called inside a transaction or tight lock.
	go s.persistNewsItem(item)
	return item
}

func (s *MarketStore) publishNewsItem(now time.Time, item NewsItem) NewsItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.publishNewsItemLocked(now, item)
}

func (s *MarketStore) applyNewsImpact(item NewsItem, eng *engine.Engine, rng *rand.Rand, cfg NewsEngineConfig) {
	if eng == nil || cfg.BaseQuantity <= 0 {
		return
	}
	if math.Abs(item.Sentiment) < cfg.MinConfidence {
		return
	}
	assetIDs := s.newsImpactAssets(item)
	if len(assetIDs) == 0 {
		return
	}
	rng = ensureRand(rng)
	for _, assetID := range assetIDs {
		if assetID == 0 {
			continue
		}
		s.mu.Lock()
		price := s.marketPriceLocked(assetID)
		s.ensureNewsBotsLocked()
		s.mu.Unlock()
		if price <= 0 {
			continue
		}
		impactFactor := cfg.ImpactFactor
		if cfg.ImpactJitter > 0 {
			impactFactor *= 1 + ((rng.Float64()*2)-1)*cfg.ImpactJitter
		}
		delta := float64(price) * impactFactor * item.Sentiment
		targetPrice := price + int64(math.Round(delta))
		if targetPrice < 1 {
			targetPrice = 1
		}
		quantity := int64(math.Round(float64(cfg.BaseQuantity) * math.Abs(item.Sentiment)))
		if quantity < 1 {
			continue
		}
		s.mu.Lock()
		s.ensureNewsBalancesLocked(assetID, quantity, targetPrice, item.Sentiment < 0)
		s.mu.Unlock()
		makerSide := engine.SideSell
		takerSide := engine.SideBuy
		if item.Sentiment < 0 {
			makerSide = engine.SideBuy
			takerSide = engine.SideSell
		}
		maker := &engine.Order{
			UserID:   newsLiquidityUserID,
			AssetID:  assetID,
			Side:     makerSide,
			Type:     engine.OrderTypeLimit,
			Quantity: quantity,
			Price:    targetPrice,
		}
		taker := &engine.Order{
			UserID:   newsReactorUserID,
			AssetID:  assetID,
			Side:     takerSide,
			Type:     engine.OrderTypeMarket,
			Quantity: quantity,
		}
		ctx, cancel := context.WithTimeout(context.Background(), newsOrderTimeout)
		result, err := eng.SubmitOrder(ctx, maker)
		cancel()
		if err != nil {
			continue
		}
		ctx, cancel = context.WithTimeout(context.Background(), newsOrderTimeout)
		_, _ = eng.SubmitOrder(ctx, taker)
		cancel()
		if result.Order == nil {
			continue
		}
		ctx, cancel = context.WithTimeout(context.Background(), newsOrderTimeout)
		_, _ = eng.CancelOrder(ctx, assetID, result.Order.ID)
		cancel()
	}
}

func (s *MarketStore) ensureNewsBotsLocked() {
	ensureBot := func(userID int64, name string) {
		user := s.EnsureUser(userID)
		if user.Role != "bot" {
			user.Role = "bot"
			if strings.TrimSpace(name) != "" {
				user.Username = name
			}
			ctx, cancel := s.dbContext()
			defer cancel()
			_ = s.queries.UpsertUser(ctx, user, time.Now().UTC())
		}
	}
	ensureBot(newsReactorUserID, "news-reactor")
	ensureBot(newsLiquidityUserID, "news-impact")
}

func (s *MarketStore) ensureNewsBalancesLocked(assetID, quantity, price int64, bearish bool) {
	if assetID == 0 || quantity <= 0 || price <= 0 {
		return
	}
	requiredCash, ok := safeMultiplyInt64(price, quantity)
	if !ok {
		requiredCash = price
	}
	cashBuffer, ok := safeMultiplyInt64(requiredCash, newsCashBufferMultiplier)
	if !ok {
		cashBuffer = requiredCash
	}
	s.EnsureUser(newsLiquidityUserID)
	s.EnsureUser(newsReactorUserID)
	if bearish {
		// Maker buys, taker sells.
		if s.GetBalance(newsLiquidityUserID, defaultCurrency) < cashBuffer {
			s.SetBalance(newsLiquidityUserID, defaultCurrency, cashBuffer)
		}
		if s.GetPosition(newsReactorUserID, assetID) < quantity {
			s.SetPosition(newsReactorUserID, assetID, quantity)
		}
	} else {
		// Maker sells, taker buys.
		if s.GetPosition(newsLiquidityUserID, assetID) < quantity {
			s.SetPosition(newsLiquidityUserID, assetID, quantity)
		}
		if s.GetBalance(newsReactorUserID, defaultCurrency) < cashBuffer {
			s.SetBalance(newsReactorUserID, defaultCurrency, cashBuffer)
		}
	}
}

func (s *MarketStore) newsImpactAssets(item NewsItem) []int64 {
	assets := s.Assets(AssetFilter{})
	if len(assets) == 0 {
		return nil
	}
	ids := make(map[int64]struct{})
	if item.AssetID != 0 {
		ids[item.AssetID] = struct{}{}
	}
	for _, scope := range item.ImpactScope {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		upperScope := strings.ToUpper(scope)
		switch upperScope {
		case "ALL_STOCKS":
			for _, asset := range assets {
				if stringsEqualFold(asset.Type, "STOCK") {
					ids[asset.ID] = struct{}{}
				}
			}
			continue
		case "BOND_YIELDS":
			for _, asset := range assets {
				if stringsEqualFold(asset.Type, "BOND") {
					ids[asset.ID] = struct{}{}
				}
			}
			continue
		}
		sector := upperScope
		if strings.HasSuffix(sector, "_SECTOR") {
			sector = strings.TrimSuffix(sector, "_SECTOR")
		}
		for _, asset := range assets {
			if stringsEqualFold(asset.Symbol, scope) || stringsEqualFold(asset.Sector, sector) {
				ids[asset.ID] = struct{}{}
			}
		}
	}
	if len(ids) == 0 {
		for _, asset := range assets {
			if stringsEqualFold(asset.Type, "STOCK") {
				ids[asset.ID] = struct{}{}
			}
		}
	}
	resolved := make([]int64, 0, len(ids))
	for id := range ids {
		resolved = append(resolved, id)
	}
	sort.Slice(resolved, func(i, j int) bool { return resolved[i] < resolved[j] })
	return resolved
}
