// Package tui provides minimal terminal output formatting without a full TUI.
package tui

import (
	"fmt"
	"io"
	"strings"
)

// ANSI color codes.
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

// Printer formats output with optional color.
type Printer struct {
	out      io.Writer
	err      io.Writer
	useColor bool
}

// NewPrinter creates a new output printer.
func NewPrinter(stdout, stderr io.Writer, useColor bool) *Printer {
	return &Printer{out: stdout, err: stderr, useColor: useColor}
}

func (p *Printer) color(c string, text string) string {
	if !p.useColor {
		return text
	}
	return c + text + Reset
}

// Status prints a status line to stderr.
func (p *Printer) Status(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.err, p.color(Dim, "  "+msg))
}

// Info prints an info message.
func (p *Printer) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.err, p.color(Blue, "● "+msg))
}

// Success prints a success message.
func (p *Printer) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.err, p.color(Green, "✓ "+msg))
}

// Warn prints a warning.
func (p *Printer) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.err, p.color(Yellow, "⚠ "+msg))
}

// Error prints an error.
func (p *Printer) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.err, p.color(Red, "✗ "+msg))
}

// Text outputs text to stdout.
func (p *Printer) Text(s string) {
	fmt.Fprint(p.out, s)
}

// ToolCall prints a tool call status.
func (p *Printer) ToolCall(name, args string) {
	args = truncateStr(args, 80)
	fmt.Fprintln(p.err, p.color(Cyan, "  🔧 "+name+" "+args))
}

// ToolResult prints a tool result status.
func (p *Printer) ToolResult(name, result string) {
	result = truncateStr(result, 120)
	lines := strings.Split(result, "\n")
	if len(lines) > 0 {
		fmt.Fprintln(p.err, p.color(Gray, "    "+lines[0]))
	}
}

// Thinking prints a thinking indicator.
func (p *Printer) Thinking() {
	fmt.Fprint(p.err, p.color(Dim, "  thinking..."))
}

// Separator prints a horizontal rule to stderr.
func (p *Printer) Separator() {
	fmt.Fprintln(p.err, p.color(Dim, strings.Repeat("─", 40)))
}

// ModelHeader prints the model being used.
func (p *Printer) ModelHeader(provider, model string) {
	fmt.Fprintln(p.err, p.color(Bold+Cyan, fmt.Sprintf("pi ● %s/%s", provider, model)))
	p.Separator()
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
