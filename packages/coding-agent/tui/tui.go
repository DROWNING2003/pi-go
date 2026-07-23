// Package tui implements the interactive terminal UI using Bubble Tea.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DROWNING2003/pi-go/packages/agent/loop"
	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/protocol"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// --- Messages ---
type streamDeltaMsg struct{ text string }
type streamDoneMsg struct {
	fullText string
	err      error
}

// --- Styles ---
var (
	styUser   = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	styAsst   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	styMeta   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styStream = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styTool   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

type chatLine struct{ role, content string }

type Model struct {
	viewport   viewport.Model
	textarea   textarea.Model
	ready      bool
	width      int
	height     int
	agentModel *provider.ProviderModel
	prov       *provider.ProviderConfig
	client     *protocol.HTTPClient
	tools      *tool.Registry
	cwd        string
	messages   []chatLine
	streaming  bool
	streamIdx  int
	pending    chan tea.Msg
}

func New(m *provider.ProviderModel, prov *provider.ProviderConfig, client *protocol.HTTPClient, tools *tool.Registry, cwd string) *Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (/quit to exit)"
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Focus()

	return &Model{
		viewport:   viewport.New(80, 20),
		textarea:   ta,
		agentModel: m,
		prov:       prov,
		client:     client,
		tools:      tools,
		cwd:        cwd,
		pending:    make(chan tea.Msg, 256),
		messages: []chatLine{
			{role: "meta", content: fmt.Sprintf("pi ● %s/%s — ctrl+c to quit", m.Provider, m.ID)},
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.waitPending)
}

// waitPending reads messages from the pending channel and returns them as tea.Msg.
func (m *Model) waitPending() tea.Msg {
	return <-m.pending
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-5)
			m.textarea.SetWidth(msg.Width)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 5
			m.textarea.SetWidth(msg.Width)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "enter":
			if m.streaming {
				return m, tea.Batch(m.waitPending, m.updateTextarea(msg))
			}
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}
			if input == "/quit" || input == "/exit" {
				return m, tea.Quit
			}
			m.textarea.Reset()
			m.messages = append(m.messages, chatLine{"user", input})
			m.messages = append(m.messages, chatLine{"assistant", ""})
			m.streamIdx = len(m.messages) - 1
			m.streaming = true
			m.refresh()
			return m, tea.Batch(m.waitPending, m.runAgent(input))

		case "ctrl+n":
			m.messages = []chatLine{
				{role: "meta", content: fmt.Sprintf("pi ● %s/%s — new session", m.agentModel.Provider, m.agentModel.ID)},
			}
			m.refresh()
			return m, nil
		}

	case streamDeltaMsg:
		if m.streamIdx >= 0 && m.streamIdx < len(m.messages) {
			m.messages[m.streamIdx].content += msg.text
			m.refresh()
		}
		return m, m.waitPending

	case streamDoneMsg:
		m.streaming = false
		if msg.err != nil {
			m.messages[m.streamIdx].content = "✗ " + msg.err.Error()
		}
		m.streamIdx = -1
		m.refresh()
		return m, m.waitPending
	}

	return m, m.updateTextarea(msg)
}

func (m *Model) updateTextarea(msg tea.Msg) tea.Cmd {
	m.textarea, _ = m.textarea.Update(msg)
	return nil
}

func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	var lines []string
	for _, cl := range m.messages {
		switch cl.role {
		case "user":
			lines = append(lines, styUser.Render("▸ "+cl.content))
		case "assistant":
			if m.streaming && cl.content == "" && m.streamIdx >= 0 && &m.messages[m.streamIdx] == &cl {
				lines = append(lines, styStream.Render("● ..."))
			} else {
				lines = append(lines, cl.content)
			}
		case "tool":
			lines = append(lines, styTool.Render("  🔧 "+cl.content))
		case "meta":
			lines = append(lines, styMeta.Render(cl.content))
		}
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoBottom()

	status := styMeta.Render(fmt.Sprintf("pi ● %s/%s", m.agentModel.Provider, m.agentModel.ID))
	if m.streaming {
		status += " " + styStream.Render("● streaming")
	}
	sep := styMeta.Render(strings.Repeat("─", m.width))
	return lipgloss.JoinVertical(lipgloss.Top, m.viewport.View(), sep, status, m.textarea.View())
}

func (m *Model) refresh() {}

// runAgent runs the agent loop in a goroutine and sends results through m.pending.
func (m *Model) runAgent(input string) tea.Cmd {
	return func() tea.Msg {
		go func() {
			streamFn := func(ctx context.Context, pm *provider.ProviderModel, c *provider.Context, so *provider.StreamOptions) <-chan model.StreamEvent {
				switch m.prov.API {
				case "openai-completions":
					return protocol.StreamChatCompletion(ctx, m.client, pm, c, so)
				case "anthropic-messages":
					return protocol.StreamAnthropicMessages(ctx, m.client, pm, c, so)
				case "google-generative-ai":
					return protocol.StreamGoogleGenerate(ctx, m.client, pm, c, so)
				default:
					ch := make(chan model.StreamEvent, 1)
					ch <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{ErrorMessage: "unsupported"})
					close(ch)
					return ch
				}
			}

			config := &loop.Config{
				Model:    m.agentModel,
				Tools:    m.tools,
				MaxTurns: 10,
				StreamFn: streamFn,
			}

			userMsg := &model.UserMessage{
				Role: "user", Content: model.UserContent{model.NewTextContent(input)},
				Timestamp: time.Now().UnixMilli(),
			}

			ctx := context.Background()
			messages, err := loop.Run(ctx, config, []*model.UserMessage{userMsg})
			if err != nil {
				m.pending <- streamDoneMsg{err: err}
				return
			}

			var text strings.Builder
			for _, msg := range messages {
				if msg.Assistant != nil {
					for _, block := range msg.Assistant.Content {
						if block.Type == model.ContentTypeText {
							text.WriteString(block.Text)
						}
					}
				}
			}
			m.pending <- streamDoneMsg{fullText: text.String()}
		}()
		return nil
	}
}
