package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/DROWNING2003/pi-go/packages/agent/event"
	"github.com/DROWNING2003/pi-go/packages/agent/loop"
	promptpkg "github.com/DROWNING2003/pi-go/packages/agent/prompt"
	"github.com/DROWNING2003/pi-go/packages/agent/tool"
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/protocol"

	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/storage/config"
	"github.com/DROWNING2003/pi-go/packages/storage/session"
)

type options struct {
	Print     bool
	List      bool
	Continue  bool
	RPC       bool
	Resume    string
	Model     string
	Provider  string
	System    string
	Workspace string
	Version   string
	Args      []string
}

func parseFlags(args []string) (*options, []string, error) {
	opts := &options{}
	flags := pflag.NewFlagSet("pi", pflag.ContinueOnError)
	flags.BoolVar(&opts.Print, "print", false, "Non-interactive print mode")
	flags.BoolVar(&opts.List, "list", false, "List saved sessions")
	flags.BoolVar(&opts.Continue, "continue", false, "Continue the latest session")
	flags.BoolVar(&opts.RPC, "rpc", false, "Run in JSON-RPC headless mode")
	flags.StringVar(&opts.Resume, "resume", "", "Resume a session by ID prefix")
	flags.StringVar(&opts.Model, "model", "", "Model to use (e.g. deepseek/deepseek-chat)")
	flags.StringVar(&opts.Provider, "provider", "", "Provider to use (e.g. deepseek)")
	flags.StringVar(&opts.System, "system", "", "System prompt override")
	flags.StringVar(&opts.Workspace, "workspace", "", "Workspace directory (default: current dir)")
	flags.SetOutput(io.Discard)

	if err := flags.Parse(args); err != nil {
		return opts, args, err
	}
	opts.Args = flags.Args()
	return opts, opts.Args, nil
}

func Run(args []string, stdout io.Writer, stderr io.Writer, version string) int {
	// Handle --help before flag parsing
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Fprint(stdout, usageText)
			return 0
		}
		if a == "--version" || a == "-v" {
			fmt.Fprintf(stdout, "pi %s\n", version)
			return 0
		}
	}

	opts, remaining, err := parseFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, "unknown option:", strings.TrimPrefix(err.Error(), "unknown flag: "))
		return 2
	}

	// Commands that don't need API key
	if opts.List {
		return listSessions(stdout, stderr)
	}
	if opts.Resume != "" {
		return resumeSession(stdout, stderr, opts.Resume)
	}
	if opts.RPC {
		return runRPCMode(stdout, stderr, version, opts.Model)
	}

	// Default workspace
	cwd, _ := os.Getwd()
	if opts.Workspace != "" {
		cwd = opts.Workspace
	}

	// Config dir
	configDir, _ := os.UserConfigDir()
	configDir = filepath.Join(configDir, "pi-go")

	// Load config
	cfg, _ := config.Load(cwd, configDir)

	// Resolve model
	modelRef := opts.Model
	if modelRef == "" {
		modelRef = cfg.Model
	}
	if modelRef == "" {
		modelRef = "deepseek/deepseek-chat" // default
	}

	// Setup provider registry
	reg := provider.NewRegistry(nil)
	provider.RegisterBuiltins(reg)

	m := reg.ResolveModel(modelRef)
	if m == nil {
		fmt.Fprintf(stderr, "unknown model: %s\n", modelRef)
		return 1
	}

	prov := reg.GetProvider(m.Provider)
	if prov == nil {
		fmt.Fprintf(stderr, "unknown provider: %s\n", m.Provider)
		return 1
	}

	// Resolve API key
	apiKey := reg.ResolveAPIKeyForProvider(m.Provider, "")
	if apiKey == "" && m.Provider != "faux" {
		fmt.Fprintf(stderr, "no API key for %s (set %s)\n", m.Provider, strings.Join(prov.AuthEnvVars, " or "))
		return 1
	}

	// Build HTTP headers
	headers := map[string]string{}
	switch prov.API {
	case "openai-completions", "openai-responses":
		headers["Authorization"] = "Bearer " + apiKey
	case "anthropic-messages":
		headers["x-api-key"] = apiKey
		headers["anthropic-version"] = "2023-06-01"
	case "google-generative-ai":
		headers["x-goog-api-key"] = apiKey
	}

	client := protocol.NewHTTPClient(prov.BaseURL, headers)

	// Setup tools
	tools := tool.NewRegistry()
	tools.Register(tool.NewReadTool(cwd))
	tools.Register(tool.NewWriteTool(cwd))
	tools.Register(tool.NewEditTool(cwd))
	tools.Register(tool.NewBashTool(cwd))
	tools.Register(tool.NewWebFetchTool())

	// Load previous session for --continue
	var prevMsgs []json.RawMessage
	if opts.Continue {
		sessionsDir := filepath.Join(configDir, "sessions")
		latest, err := session.Latest(sessionsDir)
		if err == nil && latest != nil {
			prevMsgs = latest.Entries
			fmt.Fprintf(stderr, "continuing session %s (%d messages)...\n", latest.Header.ID, len(latest.Entries))
		}
	}

	// Print mode
	if opts.Print || len(remaining) > 0 {
		return runPrintMode(stdout, stderr, m, prov, client, tools, cfg, opts, remaining, cwd, prevMsgs)
	}

	if !opts.Print && len(remaining) == 0 && !opts.RPC {
		return runPiTUI(args)
	}
	return 1
}

func runPiTUI(args []string) int {
	cmd := exec.Command("pi", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func runPrintMode(stdout, stderr io.Writer, m *provider.ProviderModel, prov *provider.ProviderConfig, client *protocol.HTTPClient, tools *tool.Registry, cfg *config.Config, opts *options, args []string, cwd string, prevMsgs []json.RawMessage) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "no prompt provided")
		return 1
	}

	prompt := strings.Join(args, " ")
	fmt.Fprintf(stderr, "using %s/%s...\n", m.Provider, m.ID)

	systemPrompt := opts.System
	if systemPrompt == "" {
		systemPrompt = cfg.SystemPrompt
	}
	if systemPrompt == "" {
		// Build default system prompt with project context
		homeDir, _ := os.UserHomeDir()
		contextFiles, _ := config.LoadContextFiles(cwd, homeDir)
		skillsDir := filepath.Join(homeDir, ".agents", "skills")
		skills := promptpkg.LoadSkills(skillsDir)
		systemPrompt = promptpkg.Build(promptpkg.Options{
			Base:         promptpkg.DefaultBase(),
			ContextFiles: contextFiles,
			Skills:       skills,
			ToolNames:    promptpkg.ToolNames(tools),
		})
	}

	streamFn := func(ctx context.Context, pm *provider.ProviderModel, c *provider.Context, so *provider.StreamOptions) <-chan model.StreamEvent {
		switch prov.API {
		case "openai-completions":
			return protocol.StreamChatCompletion(ctx, client, pm, c, so)
		case "openai-responses":
			return protocol.StreamOpenAIResponses(ctx, client, pm, c, so)
		case "anthropic-messages":
			return protocol.StreamAnthropicMessages(ctx, client, pm, c, so)
		case "google-generative-ai":
			return protocol.StreamGoogleGenerate(ctx, client, pm, c, so)
		default:
			ch := make(chan model.StreamEvent, 1)
			ch <- model.NewErrorEvent(model.StopReasonError, &model.AssistantMessage{ErrorMessage: "unsupported API: " + prov.API})
			close(ch)
			return ch
		}
	}

	// Inject previous session messages into stream context
	baseStreamFn := streamFn
	if len(prevMsgs) > 0 {
		streamFn = func(ctx context.Context, pm *provider.ProviderModel, c *provider.Context, so *provider.StreamOptions) <-chan model.StreamEvent {
			allMsgs := make([]json.RawMessage, 0, len(prevMsgs)+len(c.Messages))
			allMsgs = append(allMsgs, prevMsgs...)
			allMsgs = append(allMsgs, c.Messages...)
			c.Messages = allMsgs
			return baseStreamFn(ctx, pm, c, so)
		}
	}

	config := &loop.Config{
		Model:        m,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		MaxTurns:     10,
		StreamFn:     streamFn,
	}

	userMsg := &model.UserMessage{
		Role:      "user",
		Content:   model.UserContent{model.NewTextContent(prompt)},
		Timestamp: 0, // will be set by agent loop
	}

	fmt.Fprintln(stderr, "")

	ctx := context.Background()
	messages, err := loop.Run(ctx, config, []*model.UserMessage{userMsg})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Print all assistant responses
	for _, msg := range messages {
		if msg.Assistant != nil {
			for _, block := range msg.Assistant.Content {
				switch block.Type {
				case model.ContentTypeText:
					fmt.Fprint(stdout, block.Text)
				case model.ContentTypeThinking:
					fmt.Fprintf(stderr, "[thinking] %s\n", block.Thinking)
				case model.ContentTypeToolCall:
					fmt.Fprintf(stderr, "[tool:%s] %s\n", block.Name, string(block.Arguments))
				}
			}
		}
		if msg.ToolResult != nil {
			text := ""
			for _, b := range msg.ToolResult.Content {
				if b.Type == model.ContentTypeText {
					text += b.Text
				}
			}
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			fmt.Fprintf(stderr, "[%s] %s\n", msg.ToolResult.ToolName, text)
		}
	}
	fmt.Fprintln(stdout)

	// Save session
	saveSession(messages, cwd)

	return 0
}

func saveSession(messages []event.Message, cwd string) {
	d := configDir()
	sessionsDir := filepath.Join(d, "sessions")
	os.MkdirAll(sessionsDir, 0700)
	id := fmt.Sprintf("print-%d", time.Now().Unix())
	s := session.New(id, cwd)
	s.SetPath(filepath.Join(sessionsDir, id+".jsonl"))
	for _, msg := range messages {
		data, _ := json.Marshal(msg)
		s.Append(data)
	}
	_ = s.Save()
}

func configDir() string {
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "pi-go")
}

const usageText = `Usage: pi [options] [message...]

Options:
  --help       Show help
  --version    Show version
  --print      Non-interactive print mode (default when message provided)
  --model      Model reference (e.g. deepseek/deepseek-chat, openai/gpt-4o)
  --provider   Provider override
  --system     System prompt override
  --workspace  Workspace directory (default: current dir)

Examples:
  pi --print "What is Go?"
  pi --model anthropic/claude-sonnet-4-6 --print "Explain concurrency"
  pi --system "Be concise" --print "Hello"

Environment:
  DEEPSEEK_API_KEY   DeepSeek API key
  OPENAI_API_KEY     OpenAI API key
  ANTHROPIC_API_KEY  Anthropic API key
  GOOGLE_API_KEY     Google API key
`

func listSessions(stdout, stderr io.Writer) int {
	sessionsDir := filepath.Join(configDir(), "sessions")
	sessions, err := session.List(sessionsDir)
	if err != nil {
		fmt.Fprintf(stderr, "list sessions: %v\n", err)
		return 1
	}
	if len(sessions) == 0 {
		fmt.Fprintln(stdout, "No saved sessions.")
		return 0
	}
	fmt.Fprintf(stdout, "%-20s  %-20s  %s\n", "ID", "TIME", "MESSAGES")
	for _, s := range sessions {
		fmt.Fprintf(stdout, "%-20s  %-20s  %d\n", s.ID, s.Timestamp, s.Entries)
	}
	return 0
}

func resumeSession(stdout, stderr io.Writer, idPrefix string) int {
	sessionsDir := filepath.Join(configDir(), "sessions")
	s, err := session.FindByID(sessionsDir, idPrefix)
	if err != nil || s == nil {
		fmt.Fprintf(stderr, "session not found: %s\n", idPrefix)
		return 1
	}
	fmt.Fprintf(stderr, "resuming session %s (%d messages)...\n", s.Header.ID, len(s.Entries))
	return 0
}
