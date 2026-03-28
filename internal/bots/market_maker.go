package bots

import (
	"math"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

type Quote struct {
	BidPrice int64
	AskPrice int64
}

const bpsDivisor int64 = 10000

func MidPrice(snapshot engine.OrderBookSnapshot, fallback int64) int64 {
	var bid int64
	var ask int64
	if len(snapshot.Bids) > 0 {
		bid = snapshot.Bids[0].Price
	}
	if len(snapshot.Asks) > 0 {
		ask = snapshot.Asks[0].Price
	}
	switch {
	case bid > 0 && ask > 0:
		return (bid + ask) / 2
	case snapshot.LastPrice > 0:
		return snapshot.LastPrice
	case bid > 0:
		return bid
	case ask > 0:
		return ask
	case fallback > 0:
		return fallback
	default:
		return 1
	}
}

func QuoteFromMid(mid int64, spreadBps int64) Quote {
	if mid <= 0 {
		mid = 1
	}
	if spreadBps <= 0 {
		spreadBps = 1
	}
	spread := int64(math.Round(float64(mid) * float64(spreadBps) / float64(bpsDivisor)))
	if spread < 1 {
		spread = 1
	}
	half := spread / 2
	bid := mid - half
	ask := mid + (spread - half)
	if bid < 1 {
		bid = 1
	}
	if ask <= bid {
		ask = bid + 1
	}
	return Quote{BidPrice: bid, AskPrice: ask}
}

func QuoteFromSnapshot(snapshot engine.OrderBookSnapshot, spreadBps int64, fallback int64) Quote {
	return QuoteFromMid(MidPrice(snapshot, fallback), spreadBps)
}
