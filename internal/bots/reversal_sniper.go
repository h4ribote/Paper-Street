package bots

type ReversalSignal struct {
	Side string
}

func ReversalSignalFromCandles(candles []Candle, rsiPeriod int, sigma float64) (ReversalSignal, bool) {
	if len(candles) == 0 {
		return ReversalSignal{}, false
	}
	closes := ClosingPrices(candles)
	rsi := RSI(closes, rsiPeriod)
	_, upper, lower, ok := Bollinger(closes, rsiPeriod, sigma)
	if !ok {
		return ReversalSignal{}, false
	}
	last := float64(closes[len(closes)-1])
	switch {
	case last > upper && rsi >= 90:
		return ReversalSignal{Side: "SELL"}, true
	case last < lower && rsi <= 10:
		return ReversalSignal{Side: "BUY"}, true
	default:
		return ReversalSignal{}, false
	}
}
