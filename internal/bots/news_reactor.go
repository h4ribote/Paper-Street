package bots

import "math"

type NewsEvent struct {
	AssetID    int64   `json:"asset_id"`
	Sentiment  float64 `json:"sentiment"`
	Confidence float64 `json:"confidence"`
	Headline   string  `json:"headline,omitempty"`
}

type Reaction struct {
	Side     string
	Quantity int64
}

func ReactionOrder(event NewsEvent, baseQuantity int64, minConfidence float64) (Reaction, bool) {
	if baseQuantity <= 0 {
		return Reaction{}, false
	}
	if event.Confidence < minConfidence {
		return Reaction{}, false
	}
	score := event.Sentiment * event.Confidence
	if score == 0 {
		return Reaction{}, false
	}
	quantity := int64(math.Round(float64(baseQuantity) * math.Abs(score)))
	if quantity < 1 {
		return Reaction{}, false
	}
	side := "BUY"
	if score < 0 {
		side = "SELL"
	}
	return Reaction{Side: side, Quantity: quantity}, true
}
