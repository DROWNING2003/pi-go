package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSEParser_BasicEvents(t *testing.T) {
	input := `data: {"chunk":1}

data: {"chunk":2}

data: [DONE]
`
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.Next()
	if err != nil {
		t.Fatalf("event 1: %v", err)
	}
	if event.Data != `{"chunk":1}` {
		t.Errorf("event 1 data: %q", event.Data)
	}

	event, err = parser.Next()
	if err != nil {
		t.Fatalf("event 2: %v", err)
	}
	if event.Data != `{"chunk":2}` {
		t.Errorf("event 2 data: %q", event.Data)
	}

	event, err = parser.Next()
	if err != nil {
		t.Fatalf("event 3: %v", err)
	}
	if event.Data != "[DONE]" {
		t.Errorf("event 3 data: %q", event.Data)
	}

	_, err = parser.Next()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestSSEParser_MultiLineData(t *testing.T) {
	input := `data: line1
data: line2
data: line3

data: end
`
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.Next()
	if err != nil {
		t.Fatalf("event: %v", err)
	}
	if event.Data != "line1\nline2\nline3" {
		t.Errorf("multiline data: %q", event.Data)
	}
}

func TestSSEParser_WithEventType(t *testing.T) {
	input := `event: update
data: {"text":"hi"}

event: done
data: {"status":"ok"}
`
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.Next()
	if err != nil {
		t.Fatalf("event 1: %v", err)
	}
	if event.Event != "update" || event.Data != `{"text":"hi"}` {
		t.Errorf("event 1: event=%q data=%q", event.Event, event.Data)
	}

	event, err = parser.Next()
	if err != nil {
		t.Fatalf("event 2: %v", err)
	}
	if event.Event != "done" {
		t.Errorf("event 2 event: %q", event.Event)
	}
}

func TestSSEParser_CRLF(t *testing.T) {
	input := "data: hello\r\n\r\ndata: world\r\n\r\n"
	parser := NewSSEParser(strings.NewReader(input))

	event, _ := parser.Next()
	if event.Data != "hello" {
		t.Errorf("first: %q", event.Data)
	}
	event, _ = parser.Next()
	if event.Data != "world" {
		t.Errorf("second: %q", event.Data)
	}
}

func TestSSEParser_CommentsIgnored(t *testing.T) {
	input := `: this is a comment
data: real data
: another comment

`
	parser := NewSSEParser(strings.NewReader(input))

	event, err := parser.Next()
	if err != nil {
		t.Fatalf("event: %v", err)
	}
	if event.Data != "real data" {
		t.Errorf("data: %q", event.Data)
	}
}

func TestHTTPClient_DoStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: ok\n\n")
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, nil)
	resp, err := client.DoStream(context.Background(), "/v1/chat", map[string]string{"test": "1"}, nil)
	if err != nil {
		t.Fatalf("DoStream: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "data: ok\n\n" {
		t.Errorf("body: %q", body)
	}
}

func TestHTTPClient_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"invalid key"}`)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, nil)
	_, err := client.DoStream(context.Background(), "/", nil, nil)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 401 {
		t.Errorf("status: %d", httpErr.StatusCode)
	}
}

func TestHTTPClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewHTTPClient(server.URL, nil)
	_, err := client.DoStream(ctx, "/", nil, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSanitizeHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer sk-secret1234key",
		"Content-Type":  "application/json",
		"x-api-key":     "abcdefgh12345678",
	}

	sanitized := SanitizeHeaders(headers)

	if !strings.Contains(sanitized["Authorization"], "...") {
		t.Errorf("auth not sanitized: %s", sanitized["Authorization"])
	}
	if sanitized["Content-Type"] != "application/json" {
		t.Errorf("non-sensitive header changed: %s", sanitized["Content-Type"])
	}
	if sanitized["x-api-key"] == headers["x-api-key"] {
		t.Errorf("api key not sanitized: %s", sanitized["x-api-key"])
	}
}

func TestJSONLinesParser(t *testing.T) {
	input := `{"a":1}
{"b":2}

{"c":3}
`
	parser := NewJSONLinesParser(strings.NewReader(input))

	var results []json.RawMessage
	for {
		msg, err := parser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		results = append(results, msg)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(results))
	}
	if string(results[0]) != `{"a":1}` {
		t.Errorf("line 1: %s", results[0])
	}
	if string(results[2]) != `{"c":3}` {
		t.Errorf("line 3: %s", results[2])
	}
}
