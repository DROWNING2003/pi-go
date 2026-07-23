package util

import (
	"crypto/rand"
	"fmt"
	"math"
	"time"
)

// UUIDv7 generates a time-ordered UUID v7.
func UUIDv7() string {
	b := make([]byte, 16)
	rand.Read(b)

	now := uint64(time.Now().UnixMilli())
	b[0] = byte(now >> 40)
	b[1] = byte(now >> 32)
	b[2] = byte(now >> 24)
	b[3] = byte(now >> 16)
	b[4] = byte(now >> 8)
	b[5] = byte(now)

	b[6] = (b[6] & 0x0f) | 0x70
	b[7] = b[7] & 0x3f
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// EstimateTokens estimates token count from text length.
func EstimateTokens(text string) int {
	return int(math.Ceil(float64(len(text)) / 4.0))
}

// ShortHash generates a short hex hash from a string.
func ShortHash(s string) string {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return fmt.Sprintf("%08x", h%0xffffffff)
}
