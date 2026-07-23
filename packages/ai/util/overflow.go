package util

// ContextOverflow truncates old messages when total tokens exceed the context window.
// Returns the truncated message list and the number of tokens removed.
func ContextOverflow(messages [][]byte, maxTokens int) ([][]byte, int) {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(string(m))
	}
	if total <= maxTokens {
		return messages, 0
	}

	// Remove oldest non-system messages first
	removed := 0
	var result [][]byte
	for i := len(messages) - 1; i >= 0; i-- {
		tok := EstimateTokens(string(messages[i]))
		if total-tok <= maxTokens {
			result = append([][]byte{messages[i]}, result...)
		} else {
			removed += tok
		}
		total -= tok
	}
	return result, removed
}
