package bots

import "math"

type DipBuySignal struct {
	BuyPrice   int64
	TakeProfit int64
}

func DipBuySignalFromCandles(candles []Candle, trendShort, trendLong, bandPeriod int, sigma float64) (DipBuySignal, bool) {
	if len(candles) == 0 {
		return DipBuySignal{}, false
	}
	closes := ClosingPrices(candles)
	shortEMA := EMA(closes, trendShort)
	longEMA := EMA(closes, trendLong)
	if shortEMA <= longEMA {
		return DipBuySignal{}, false
	}
	mid, upper, lower, ok := Bollinger(closes, bandPeriod, sigma)
	if !ok {
		return DipBuySignal{}, false
	}
	lastClose := float64(closes[len(closes)-1])
	if lastClose > mid {
		return DipBuySignal{}, false
	}
	buy := int64(math.Round(mid))
	if lastClose < lower {
		buy = int64(math.Round(lower))
	}
	recentHigh := HighestHigh(candles, bandPeriod)
	recentLow := LowestLow(candles, bandPeriod)
	rangeSize := float64(recentHigh - recentLow)
	takeProfit := recentHigh
	if rangeSize > 0 {
		extension := float64(recentHigh) + rangeSize*0.618
		if extension > float64(recentHigh) {
			takeProfit = int64(math.Round(extension))
		}
	}
	if takeProfit <= buy {
		takeProfit = int64(math.Round(upper))
	}
	if buy <= 0 {
		return DipBuySignal{}, false
	}
	return DipBuySignal{BuyPrice: buy, TakeProfit: takeProfit}, true
}
