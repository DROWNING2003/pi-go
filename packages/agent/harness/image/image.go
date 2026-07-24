// Package image provides image detection utilities matching TS harness/tools/image.ts
package image

import (
	"encoding/base64"
	"strings"
)

// DetectMimeType detects the MIME type from image bytes.
func DetectMimeType(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		if len(data) > 3 && data[3] == 0xF7 {
			return ""
		}
		return "image/jpeg"
	}
	// PNG
	if len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return "image/png"
	}
	// GIF
	if strings.HasPrefix(string(data), "GIF") {
		return "image/gif"
	}
	// WebP
	if strings.HasPrefix(string(data), "RIFF") && len(data) >= 12 && strings.HasPrefix(string(data[8:]), "WEBP") {
		return "image/webp"
	}
	// BMP
	if data[0] == 'B' && data[1] == 'M' {
		return "image/bmp"
	}
	return ""
}

// EncodeBase64 encodes bytes to base64 string.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
