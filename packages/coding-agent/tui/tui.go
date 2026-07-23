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

// Model is the top-level Bubble Tea model.
type Model struct {
	viewport    viewport.Model
	textarea    textarea.Model
	messages    []string
	streaming   bool
	providerStr string
	modelStr    string
	err         error

	// Agent config
	agentModel *provider.ProviderModel
	prov       *provider.ProviderConfig
	client     *protocol.HTTPClient
	tools      *tool.Registry
	cwd        string

	width  int
	height int
	ready  bool
}

// New creates a new TUI model.
func New(m *provider.ProviderModel, prov *provider.ProviderConfig, client *protocol.HTTPClient, tools *tool.Registry, cwd string) *Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Ctrl+D or /quit to exit)"
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to pi ● " + m.Provider + "/" + m.ID + "\n\n")

	return &Model{
		viewport:    vp,
		textarea:    ta,
		providerStr: m.Provider,
		modelStr:    m.ID,
		agentModel:  m,
		prov:        prov,
		client:      client,
		tools:       tools,
		cwd:         cwd,
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.textarea.SetWidth(msg.Width)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
			m.textarea.SetWidth(msg.Width)
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.streaming {
				return m, nil
			}
			input := strings.TrimSpace(m.textarea.Value())
			if input == "/quit" || input == "/exit" {
				return m, tea.Quit
			}
			if input == "" {
				return m, nil
			}
			m.textarea.Reset()
			m.messages = append(m.messages, "> "+input)
			m.updateViewport()
			return m, m.sendPrompt(input)
		}

	case streamMsg:
		m.streaming = false
		if msg.err != nil {
			m.messages = append(m.messages, "✗ "+msg.err.Error())
		} else {
			m.messages = append(m.messages, msg.text)
		}
		m.updateViewport()
	}

	m.textarea, taCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, tea.Batch(taCmd, vpCmd)
}

// View renders the UI.
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("pi ● %s/%s", m.providerStr, m.modelStr))

	if m.streaming {
		status += " " + lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("● streaming...")
	}

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.viewport.View(),
		sep,
		status,
		m.textarea.View(),
	)
}

func (m *Model) updateViewport() {
	m.viewport.SetContent(strings.Join(m.messages, "\n"))
	m.viewport.GotoBottom()
}

// streamMsg is sent when a prompt completes.
type streamMsg struct {
	text string
	err  error
}

func (m *Model) sendPrompt(input string) tea.Cmd {
	return func() tea.Msg {
		m.streaming = true

		streamFn := func(ctx context.Context, pm *provider.ProviderModel, c *provider.Context, so *provider.StreamOptions) <-chan model.StreamEvent {
			switch m.prov.API {
			case "openai-completions":
				return protocol.StreamChatCompletion(ctx, m.client, pm, c, so)
			case "openai-responses":
				return protocol.StreamOpenAIResponses(ctx, m.client, pm, c, so)
			case "anthropic-messages":
				return protocol.StreamAnthropicMessages(ctx, m.client, pm, c, so)
			case "google-generative-ai":
				return protocol.StreamGoogleGenerate(ctx, m.client, pm, c, so)
			default:
				ch := make(chan model.StreamEvent, 1)
				ch <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{ErrorMessage: "unsupported API"})
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
			return streamMsg{err: err}
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
		return streamMsg{text: text.String()}
	}
}
