package bots

import "math"

type GridLevel struct {
	Side  string
	Price int64
}

func GridLevels(mid int64, stepBps int64, levels int) []GridLevel {
	if mid <= 0 || stepBps <= 0 || levels <= 0 {
		return nil
	}
	step := int64(math.Round(float64(mid) * float64(stepBps) / float64(bpsDivisor)))
	if step < 1 {
		step = 1
	}
	grids := make([]GridLevel, 0, levels*2)
	for i := 1; i <= levels; i++ {
		offset := step * int64(i)
		buyPrice := mid - offset
		sellPrice := mid + offset
		if buyPrice > 0 {
			grids = append(grids, GridLevel{Side: "BUY", Price: buyPrice})
		}
		grids = append(grids, GridLevel{Side: "SELL", Price: sellPrice})
	}
	return grids
}
