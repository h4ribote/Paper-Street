package bots

import "math"

func ImpactAdjustedPrice(mid int64, sigma float64, orderSize int64, volume int64, gamma float64, side string) int64 {
	if mid <= 0 {
		return 0
	}
	if orderSize <= 0 || volume <= 0 || gamma <= 0 || sigma <= 0 {
		return mid
	}
	impact := gamma * sigma * math.Sqrt(float64(orderSize)/float64(volume))
	if impact < 0 {
		impact = 0
	}
	if side == "BUY" {
		price := int64(math.Round(float64(mid) * (1 + impact)))
		if impact > 0 && price <= mid {
			price = mid + 1
		}
		return price
	}
	price := int64(math.Round(float64(mid) * (1 - impact)))
	if impact > 0 && price >= mid {
		price = mid - 1
		if price < 1 {
			price = 1
		}
	}
	return price
}

func VWAP(candles []Candle) int64 {
	if len(candles) == 0 {
		return 0
	}
	var total float64
	var volume float64
	for _, candle := range candles {
		typical := float64(candle.High+candle.Low+candle.Close) / 3
		vol := float64(candle.Volume)
		total += typical * vol
		volume += vol
	}
	if volume == 0 {
		return int64(math.Round(SMA(ClosingPrices(candles))))
	}
	return int64(math.Round(total / volume))
}
