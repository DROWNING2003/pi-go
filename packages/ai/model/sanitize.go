package model

import "strings"

// SanitizeSurrogates removes unpaired Unicode surrogate characters.
func SanitizeSurrogates(text string) string {
	var result strings.Builder
	for i, r := range text {
		if r >= 0xD800 && r <= 0xDFFF {
			// High surrogate: must be followed by low surrogate
			if r <= 0xDBFF {
				if i+1 < len(text) {
					next := rune(text[i+1])
					if next >= 0xDC00 && next <= 0xDFFF {
						result.WriteRune(r)
						continue
					}
				}
				// Unpaired high surrogate - skip
				continue
			}
			// Low surrogate: must be preceded by high surrogate
			if i > 0 {
				prev := rune(text[i-1])
				if prev >= 0xD800 && prev <= 0xDBFF {
					result.WriteRune(r)
					continue
				}
			}
			// Unpaired low surrogate - skip
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
