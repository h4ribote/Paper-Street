package bots

type MomentumSignal struct {
	Side     string
	Strength float64
}

func MomentumSignalFromCandles(candles []Candle, shortPeriod, longPeriod, breakoutLookback int, volumeMultiplier, trendThreshold float64) (MomentumSignal, bool) {
	if len(candles) == 0 {
		return MomentumSignal{}, false
	}
	closes := ClosingPrices(candles)
	shortEMA := EMA(closes, shortPeriod)
	longEMA := EMA(closes, longPeriod)
	if shortEMA == 0 || longEMA == 0 {
		return MomentumSignal{}, false
	}
	if breakoutLookback <= 1 {
		breakoutLookback = 2
	}
	if len(candles) <= breakoutLookback {
		return MomentumSignal{}, false
	}
	lookbackCandles := candles[:len(candles)-1]
	breakoutHigh := HighestHigh(lookbackCandles, breakoutLookback)
	breakoutLow := LowestLow(lookbackCandles, breakoutLookback)
	lastClose := closes[len(closes)-1]
	strength := TrendStrength(closes, longPeriod)
	if strength < trendThreshold {
		return MomentumSignal{}, false
	}
	if !VolumeSpike(candles, breakoutLookback, volumeMultiplier) {
		return MomentumSignal{}, false
	}
	switch {
	case float64(lastClose) > float64(breakoutHigh) && shortEMA > longEMA:
		return MomentumSignal{Side: "BUY", Strength: strength}, true
	case float64(lastClose) < float64(breakoutLow) && shortEMA < longEMA:
		return MomentumSignal{Side: "SELL", Strength: strength}, true
	default:
		return MomentumSignal{}, false
	}
}
