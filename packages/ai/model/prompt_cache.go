package model

const OpenAIPromptCacheKeyMaxLen = 64

// ClampPromptCacheKey truncates a session ID to OpenAI's 64-char limit.
func ClampPromptCacheKey(key string) string {
	chars := []rune(key)
	if len(chars) <= OpenAIPromptCacheKeyMaxLen {
		return key
	}
	return string(chars[:OpenAIPromptCacheKeyMaxLen])
}
