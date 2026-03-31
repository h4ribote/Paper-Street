package bots

import (
	"math"
)

func ClosingPrices(candles []Candle) []int64 {
	prices := make([]int64, 0, len(candles))
	for _, candle := range candles {
		prices = append(prices, candle.Close)
	}
	return prices
}

func SMA(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += float64(value)
	}
	return sum / float64(len(values))
}

func EMA(values []int64, period int) float64 {
	if len(values) == 0 || period <= 0 {
		return 0
	}
	if len(values) < period {
		period = len(values)
	}
	k := 2.0 / (float64(period) + 1)
	ema := SMA(values[:period])
	for i := period; i < len(values); i++ {
		price := float64(values[i])
		ema = price*k + ema*(1-k)
	}
	return ema
}

func StdDev(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := SMA(values)
	var sum float64
	for _, value := range values {
		diff := float64(value) - mean
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(values)))
}

func Bollinger(values []int64, period int, sigma float64) (mid, upper, lower float64, ok bool) {
	if period <= 0 || len(values) < period {
		return 0, 0, 0, false
	}
	window := values[len(values)-period:]
	mid = SMA(window)
	deviation := StdDev(window)
	upper = mid + sigma*deviation
	lower = mid - sigma*deviation
	return mid, upper, lower, true
}

func RSI(values []int64, period int) float64 {
	if period <= 0 || len(values) <= period {
		return 0
	}
	var gains float64
	var losses float64
	start := len(values) - period
	for i := start; i < len(values); i++ {
		if i == 0 {
			continue
		}
		diff := float64(values[i] - values[i-1])
		if diff > 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}
	if gains == 0 && losses == 0 {
		return 50
	}
	if losses == 0 {
		return 100
	}
	averageGain := gains / float64(period)
	averageLoss := losses / float64(period)
	if averageLoss == 0 {
		return 100
	}
	rs := averageGain / averageLoss
	return 100 - (100 / (1 + rs))
}

func ATR(candles []Candle, period int) float64 {
	if period <= 0 || len(candles) <= period {
		return 0
	}
	var sum float64
	start := len(candles) - period
	for i := start; i < len(candles); i++ {
		if i == 0 {
			continue
		}
		high := float64(candles[i].High)
		low := float64(candles[i].Low)
		prevClose := float64(candles[i-1].Close)
		tr := math.Max(high-low, math.Max(math.Abs(high-prevClose), math.Abs(low-prevClose)))
		sum += tr
	}
	return sum / float64(period)
}

func VolumeSpike(candles []Candle, lookback int, multiplier float64) bool {
	if len(candles) == 0 || lookback <= 0 {
		return false
	}
	if len(candles) < lookback {
		lookback = len(candles)
	}
	window := candles[len(candles)-lookback:]
	var sum float64
	for _, candle := range window {
		sum += float64(candle.Volume)
	}
	avg := sum / float64(len(window))
	latest := float64(window[len(window)-1].Volume)
	return latest >= avg*multiplier
}

func TrendStrength(values []int64, period int) float64 {
	if period <= 1 || len(values) < 2 {
		return 0
	}
	if len(values) < period {
		period = len(values)
	}
	start := len(values) - period
	var sum float64
	var count int
	for i := start; i < len(values); i++ {
		if i == 0 {
			continue
		}
		prev := float64(values[i-1])
		if prev == 0 {
			continue
		}
		diff := math.Abs(float64(values[i]) - prev)
		sum += diff / prev
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func HighestHigh(candles []Candle, lookback int) int64 {
	if len(candles) == 0 {
		return 0
	}
	if lookback <= 0 || lookback > len(candles) {
		lookback = len(candles)
	}
	start := len(candles) - lookback
	var high int64
	for i := start; i < len(candles); i++ {
		if candles[i].High > high {
			high = candles[i].High
		}
	}
	return high
}

func LowestLow(candles []Candle, lookback int) int64 {
	if len(candles) == 0 {
		return 0
	}
	if lookback <= 0 || lookback > len(candles) {
		lookback = len(candles)
	}
	start := len(candles) - lookback
	low := candles[start].Low
	for i := start + 1; i < len(candles); i++ {
		if candles[i].Low < low {
			low = candles[i].Low
		}
	}
	return low
}

func RelativeVolatility(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := SMA(values)
	if mean == 0 {
		return 0
	}
	return StdDev(values) / mean
}
