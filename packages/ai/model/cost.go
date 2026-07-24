package model

// CalculateCost computes usage cost from model pricing rates.
func CalculateCost(cost ModelCost, usage *Usage) {
	inputTokens := usage.Input + usage.CacheRead + usage.CacheWrite
	rates := ModelCost{
		Input: cost.Input, Output: cost.Output,
		CacheRead: cost.CacheRead, CacheWrite: cost.CacheWrite,
	}
	matchedThreshold := -1
	for _, tier := range cost.Tiers {
		if inputTokens > tier.InputTokensAbove && tier.InputTokensAbove > matchedThreshold {
			rates.Input = tier.Input
			rates.Output = tier.Output
			rates.CacheRead = tier.CacheRead
			rates.CacheWrite = tier.CacheWrite
			matchedThreshold = tier.InputTokensAbove
		}
	}

	longWrite := 0
	if usage.CacheWrite1h != nil {
		longWrite = *usage.CacheWrite1h
	}
	shortWrite := usage.CacheWrite - longWrite

	usage.Cost.Input = (rates.Input / 1000000) * float64(usage.Input)
	usage.Cost.Output = (rates.Output / 1000000) * float64(usage.Output)
	usage.Cost.CacheRead = (rates.CacheRead / 1000000) * float64(usage.CacheRead)
	usage.Cost.CacheWrite = ((rates.CacheWrite/1000000)*float64(shortWrite) + (rates.CacheWrite/1000000)*float64(longWrite)*2)
	usage.Cost.Total = usage.Cost.Input + usage.Cost.Output + usage.Cost.CacheRead + usage.Cost.CacheWrite
}
