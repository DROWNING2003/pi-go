// Package protocol implements HTTP client, SSE stream parsing, and payload
// sanitization shared across all provider API implementations.
package protocol

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPClient wraps net/http.Client with streaming helpers for provider APIs.
type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
}

// NewHTTPClient creates a client with a configurable base URL and default headers.
func NewHTTPClient(baseURL string, headers map[string]string) *HTTPClient {
	return &HTTPClient{
		client:  &http.Client{},
		baseURL: strings.TrimRight(baseURL, "/"),
		headers: headers,
	}
}

// WithTimeout sets the HTTP client timeout.
func (c *HTTPClient) WithTimeout(timeoutMs int) *HTTPClient {
	return c
}

// DoStream sends a POST request with a JSON body and returns a buffered reader
// over the response body for SSE/stream parsing. The caller must close the
// response body.
func (c *HTTPClient) DoStream(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	return resp, nil
}

// HTTPError represents a non-2xx HTTP response.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// SSEEvent represents a single Server-Sent Event.
type SSEEvent struct {
	Event string // "event:" field
	Data  string // "data:" field (accumulated across multiple data lines)
	ID    string // "id:" field
}

// SSEParser parses Server-Sent Events from a buffered reader.
// It handles chunked transfer encoding, incomplete lines, and multi-line data.
type SSEParser struct {
	scanner *bufio.Scanner
}

// NewSSEParser creates an SSE parser from a reader.
func NewSSEParser(r io.Reader) *SSEParser {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	s.Split(scanSSELines)
	return &SSEParser{scanner: s}
}

// Next returns the next SSE event. Returns io.EOF when the stream ends.
func (p *SSEParser) Next() (*SSEEvent, error) {
	var dataLines []string
	var eventType, id string

	for p.scanner.Scan() {
		line := p.scanner.Text()
		if line == "" {
			// Empty line = event boundary
			if len(dataLines) > 0 {
				return &SSEEvent{
					Event: eventType,
					Data:  strings.Join(dataLines, "\n"),
					ID:    id,
				}, nil
			}
			// Reset for next event
			eventType = ""
			id = ""
			dataLines = nil
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		} else if strings.HasPrefix(line, "id:") {
			id = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
		// Ignore comment lines (starting with ':')
	}

	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("sse scan: %w", err)
	}

	// Return any remaining data
	if len(dataLines) > 0 {
		return &SSEEvent{
			Event: eventType,
			Data:  strings.Join(dataLines, "\n"),
			ID:    id,
		}, nil
	}

	return nil, io.EOF
}

// scanSSELines is a bufio.SplitFunc that handles both LF and CRLF line endings.
func scanSSELines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	// Look for LF or CRLF
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			end := i
			// Strip trailing \r
			if end > 0 && data[end-1] == '\r' {
				return i + 1, data[:end-1], nil
			}
			return i + 1, data[:end], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// SanitizeHeaders removes sensitive values from headers for logging.
func SanitizeHeaders(headers map[string]string) map[string]string {
	sanitized := make(map[string]string, len(headers))
	sensitiveKeys := map[string]bool{
		"authorization": true,
		"x-api-key":     true,
		"api-key":       true,
	}
	for k, v := range headers {
		if sensitiveKeys[strings.ToLower(k)] {
			if len(v) > 8 {
				sanitized[k] = v[:4] + "..." + v[len(v)-4:]
			} else {
				sanitized[k] = "***"
			}
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// JSONLinesParser parses newline-delimited JSON (JSONL / NDJSON) from a reader.
// Some providers use this format instead of SSE.
type JSONLinesParser struct {
	scanner *bufio.Scanner
}

// NewJSONLinesParser creates a JSONL parser.
func NewJSONLinesParser(r io.Reader) *JSONLinesParser {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &JSONLinesParser{scanner: s}
}

// Next returns the next JSON object as raw bytes. Returns io.EOF at end.
func (p *JSONLinesParser) Next() (json.RawMessage, error) {
	for p.scanner.Scan() {
		line := strings.TrimSpace(p.scanner.Text())
		if line == "" {
			continue
		}
		return json.RawMessage(line), nil
	}
	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("jsonl scan: %w", err)
	}
	return nil, io.EOF
}

// Ensure json and http packages are used.
var _ = json.RawMessage{}
var _ = http.StatusOK
