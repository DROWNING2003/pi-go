// Package jsonl provides strict LF-only JSON Lines framing for RPC communication.
package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// Writer wraps an io.Writer and serializes values as JSON Lines.
type Writer struct {
	w io.Writer
}

// NewWriter creates a JSONL writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write marshals a value as JSON and writes it followed by a newline.
func (w *Writer) Write(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("jsonl marshal: %w", err)
	}
	data = append(data, '\n')
	_, err = w.w.Write(data)
	return err
}

// Reader reads JSON Lines from an io.Reader.
type Reader struct {
	scanner *bufio.Scanner
}

// NewReader creates a JSONL reader that splits on LF only.
func NewReader(r io.Reader) *Reader {
	s := bufio.NewScanner(r)
	s.Split(scanLines)
	return &Reader{scanner: s}
}

// Decode reads the next JSON line into v. Returns io.EOF at end.
func (r *Reader) Decode(v interface{}) error {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return fmt.Errorf("jsonl read: %w", err)
		}
		return io.EOF
	}
	line := r.scanner.Bytes()
	if err := json.Unmarshal(line, v); err != nil {
		return fmt.Errorf("jsonl decode: %w", err)
	}
	return nil
}

// scanLines splits on LF only (strict JSONL framing).
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			// Strip trailing \r for CRLF compatibility
			end := i
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
