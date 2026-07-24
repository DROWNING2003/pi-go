// Package extension provides the extension system host matching TS extensions/.
// Extensions are subprocesses that communicate via JSON-RPC over stdin/stdout.
package extension

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

// --- Extension RPC types ---

// ExtensionCommand is sent from host to extension process.
type ExtensionCommand struct {
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data,omitempty"`
	Event json.RawMessage `json:"event,omitempty"`
}

// ExtensionResponse is received from extension process.
type ExtensionResponse struct {
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// RegisteredTool describes a tool registered by an extension.
type RegisteredTool struct {
	Name        string          `json:"name"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// --- Extension Process ---

// Process manages a single extension subprocess.
type Process struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	name    string
	path    string
	pending map[string]chan ExtensionResponse
	seq     int
	done    chan struct{}
	onEvent func(eventType string, data json.RawMessage)
}

// NewProcess spawns an extension process.
func NewProcess(extPath string) (*Process, error) {
	var cmd *exec.Cmd

	// Check what kind of extension this is
	info, err := os.Stat(extPath)
	if err != nil {
		return nil, fmt.Errorf("extension not found: %s", extPath)
	}

	name := filepath.Base(extPath)

	if info.IsDir() {
		// Directory extension: look for index.ts or index.js
		indexTS := filepath.Join(extPath, "index.ts")
		indexJS := filepath.Join(extPath, "index.js")
		if _, err := os.Stat(indexTS); err == nil {
			cmd = exec.Command("npx", "tsx", indexTS, "--mode", "extension")
		} else if _, err := os.Stat(indexJS); err == nil {
			cmd = exec.Command("node", indexJS, "--mode", "extension")
		} else {
			return nil, fmt.Errorf("no index.ts or index.js in %s", extPath)
		}
	} else {
		cmd = exec.Command("node", extPath, "--mode", "extension")
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start extension %s: %w", name, err)
	}

	p := &Process{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewScanner(stdout),
		name:    name,
		path:    extPath,
		pending: make(map[string]chan ExtensionResponse),
		done:    make(chan struct{}),
	}

	go p.readLoop()

	return p, nil
}

// readLoop reads JSON responses from the extension process.
func (p *Process) readLoop() {
	for p.stdout.Scan() {
		line := strings.TrimSpace(p.stdout.Text())
		if line == "" {
			continue
		}
		var resp ExtensionResponse
		if json.Unmarshal([]byte(line), &resp) != nil {
			continue
		}
		// Route to pending request or event handler
		if resp.Type == "event" {
			if p.onEvent != nil {
				p.onEvent(resp.Type, resp.Data)
			}
		}
	}
	close(p.done)
}

// Send sends a command to the extension and waits for a response.
func (p *Process) Send(cmd ExtensionCommand) (*ExtensionResponse, error) {
	p.mu.Lock()
	p.seq++
	id := fmt.Sprintf("req_%d", p.seq)
	ch := make(chan ExtensionResponse, 1)
	p.pending[id] = ch
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
	}()

	// Actually send as raw JSON with the ID embedded
	msg := map[string]interface{}{
		"id":   id,
		"type": cmd.Type,
	}
	if cmd.Data != nil {
		msg["data"] = json.RawMessage(cmd.Data)
	}
	data, _ := json.Marshal(msg)

	p.mu.Lock()
	_, err := fmt.Fprintln(p.stdin, string(data))
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return &resp, nil
	case <-p.done:
		return nil, fmt.Errorf("extension %s exited", p.name)
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("extension %s timeout", p.name)
	}
}

// Notify sends a one-way event to the extension.
func (p *Process) Notify(eventType string, data interface{}) {
	raw, _ := json.Marshal(data)
	msg := map[string]interface{}{
		"type":  "event",
		"event": eventType,
		"data":  json.RawMessage(raw),
	}
	encoded, _ := json.Marshal(msg)
	p.mu.Lock()
	fmt.Fprintln(p.stdin, string(encoded))
	p.mu.Unlock()
}

// Close terminates the extension process.
func (p *Process) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stdin.Close()
	p.cmd.Process.Kill()
	return nil
}

// Name returns the extension name.
func (p *Process) Name() string { return p.name }

// --- Extension Host ---

// Host manages multiple extension processes.
type Host struct {
	mu     sync.Mutex
	exts   []*Process
	tools  []RegisteredTool
	onTool func(tool RegisteredTool)
}

// NewHost creates an extension host.
func NewHost() *Host {
	return &Host{}
}

// LoadDirectory scans a directory for extensions and loads them.
func (h *Host) LoadDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		extPath := filepath.Join(dir, entry.Name())
		if err := h.Load(extPath); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "extension %s: %v\n", entry.Name(), err)
		}
	}
	return nil
}

// Load loads a single extension.
func (h *Host) Load(path string) error {
	proc, err := NewProcess(path)
	if err != nil {
		return err
	}

	// Request tool registrations
	resp, err := proc.Send(ExtensionCommand{Type: "get_tools"})
	if err != nil {
		proc.Close()
		return err
	}

	if resp.Success && resp.Data != nil {
		var tools []RegisteredTool
		if json.Unmarshal(resp.Data, &tools) == nil {
			h.mu.Lock()
			h.tools = append(h.tools, tools...)
			h.mu.Unlock()
			for _, t := range tools {
				if h.onTool != nil {
					h.onTool(t)
				}
			}
		}
	}

	// Subscribe to session events
	proc.Notify("subscribe", map[string]interface{}{
		"events": []string{"session_start", "prompt", "tool_call"},
	})

	h.mu.Lock()
	h.exts = append(h.exts, proc)
	h.mu.Unlock()

	return nil
}

// OnTool registers a callback for when extensions register tools.
func (h *Host) OnTool(fn func(tool RegisteredTool)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onTool = fn
	// Replay existing tools
	for _, t := range h.tools {
		fn(t)
	}
}

// Emit sends an event to all loaded extensions.
func (h *Host) Emit(eventType string, data interface{}) {
	h.mu.Lock()
	exts := make([]*Process, len(h.exts))
	copy(exts, h.exts)
	h.mu.Unlock()

	for _, ext := range exts {
		ext.Notify(eventType, data)
	}
}

// Close terminates all extension processes.
func (h *Host) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ext := range h.exts {
		ext.Close()
	}
	h.exts = nil
}

// Tools returns all registered tools from extensions.
func (h *Host) Tools() []RegisteredTool {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]RegisteredTool, len(h.tools))
	copy(result, h.tools)
	return result
}

// Ensure model types referenced
var _ = model.ContentBlock{}
