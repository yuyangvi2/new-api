package claude

func NormalizeCacheCreationSplit(totalTokens int, tokens5m int, tokens1h int) (int, int) {
	if tokens5m < 0 {
		tokens5m = 0
	}
	if tokens1h < 0 {
		tokens1h = 0
	}
	if totalTokens <= 0 {
		return tokens5m, tokens1h
	}
	if tokens1h > totalTokens {
		return 0, totalTokens
	}
	remaining := totalTokens - tokens1h
	if tokens5m > remaining {
		tokens5m = remaining
	}
	return totalTokens - tokens1h, tokens1h
}
