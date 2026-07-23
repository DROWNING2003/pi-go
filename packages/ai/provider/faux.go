package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

const (
	defaultFauxAPI       = "faux"
	defaultFauxProvider  = "faux"
	defaultFauxModelID   = "faux-1"
	defaultFauxModelName = "Faux Model"
	defaultFauxBaseURL   = "http://localhost:0"
	defaultMinTokenSize  = 3
	defaultMaxTokenSize  = 5
)

// FauxProvider is a scriptable in-memory provider for testing.
type FauxProvider struct {
	id           string
	models       []*ProviderModel
	minTokenSize int
	maxTokenSize int
	tokensPerSec float64

	mu               sync.Mutex
	pendingResponses []FauxResponseStep
	callCount        int
	promptCache      map[string]string
}

// FauxResponseStep is either a final AssistantMessage or a factory function.
type FauxResponseStep interface {
	isFauxResponseStep()
}

// FauxMessage is a pre-built AssistantMessage used as a response step.
type FauxMessage struct {
	Message *model.AssistantMessage
}

func (FauxMessage) isFauxResponseStep() {}

// FauxResponseFactory produces an AssistantMessage from request context.
type FauxResponseFactory func(ctx context.Context, m *ProviderModel, c *Context, opts *StreamOptions, callCount int) *model.AssistantMessage

func (FauxResponseFactory) isFauxResponseStep() {}

// FauxProviderOption configures a FauxProvider during construction.
type FauxProviderOption func(*FauxProvider)

// WithFauxID sets the provider identifier.
func WithFauxID(id string) FauxProviderOption {
	return func(p *FauxProvider) { p.id = id }
}

// WithFauxModels sets the available models.
func WithFauxModels(models ...*ProviderModel) FauxProviderOption {
	return func(p *FauxProvider) { p.models = models }
}

// WithFauxTokenSize sets min/max token size for stream chunking.
func WithFauxTokenSize(min, max int) FauxProviderOption {
	return func(p *FauxProvider) { p.minTokenSize = min; p.maxTokenSize = max }
}

// WithFauxTokensPerSecond sets stream pacing in tokens per second.
func WithFauxTokensPerSecond(tps float64) FauxProviderOption {
	return func(p *FauxProvider) { p.tokensPerSec = tps }
}

// NewFauxProvider creates a scriptable faux provider for testing.
func NewFauxProvider(opts ...FauxProviderOption) *FauxProvider {
	p := &FauxProvider{
		id:           defaultFauxProvider,
		minTokenSize: defaultMinTokenSize,
		maxTokenSize: defaultMaxTokenSize,
		promptCache:  make(map[string]string),
		models: []*ProviderModel{{
			ID:            defaultFauxModelID,
			Name:          defaultFauxModelName,
			API:           defaultFauxAPI,
			Provider:      defaultFauxProvider,
			BaseURL:       defaultFauxBaseURL,
			Reasoning:     false,
			Input:         []string{"text", "image"},
			ContextWindow: 128000,
			MaxTokens:     16384,
		}},
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// ID returns the provider identifier.
func (p *FauxProvider) ID() string { return p.id }

// Models returns the available models.
func (p *FauxProvider) Models() []*ProviderModel { return p.models }

// GetModel returns the default model or a specific model by ID.
func (p *FauxProvider) GetModel(modelID ...string) *ProviderModel {
	if len(modelID) == 0 || modelID[0] == "" {
		return p.models[0]
	}
	for _, m := range p.models {
		if m.ID == modelID[0] {
			return m
		}
	}
	return nil
}

// CallCount returns the number of times Stream has been invoked.
func (p *FauxProvider) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.callCount
}

// PendingResponseCount returns the number of queued responses remaining.
func (p *FauxProvider) PendingResponseCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pendingResponses)
}

// SetResponses replaces the response queue.
func (p *FauxProvider) SetResponses(steps ...FauxResponseStep) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pendingResponses = make([]FauxResponseStep, len(steps))
	copy(p.pendingResponses, steps)
}

// AppendResponses adds responses to the end of the queue.
func (p *FauxProvider) AppendResponses(steps ...FauxResponseStep) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pendingResponses = append(p.pendingResponses, steps...)
}

// Stream implements the Provider interface.
func (p *FauxProvider) Stream(ctx context.Context, m *ProviderModel, c *Context, opts *StreamOptions) <-chan model.StreamEvent {
	ch := make(chan model.StreamEvent, 32)

	p.mu.Lock()
	step := p.nextStep()
	p.callCount++
	callCount := p.callCount
	p.mu.Unlock()

	go func() {
		defer close(ch)

		if step == nil {
			errMsg := p.errorMessage("no more faux responses queued", m)
			ch <- model.NewErrorEvent(model.StopReasonError, errMsg)
			return
		}

		var final *model.AssistantMessage
		switch s := step.(type) {
		case FauxMessage:
			final = p.cloneMessage(s.Message, m)
		case FauxResponseFactory:
			final = s(ctx, m, c, opts, callCount)
			final = p.cloneMessage(final, m)
		}

		final = p.withUsageEstimate(final, c, opts)
		p.streamWithDeltas(ctx, ch, final)
	}()

	return ch
}

func (p *FauxProvider) nextStep() FauxResponseStep {
	if len(p.pendingResponses) == 0 {
		return nil
	}
	step := p.pendingResponses[0]
	p.pendingResponses = p.pendingResponses[1:]
	return step
}

func (p *FauxProvider) cloneMessage(msg *model.AssistantMessage, m *ProviderModel) *model.AssistantMessage {
	cloned := *msg
	cloned.API = m.API
	cloned.Provider = m.Provider
	cloned.Model = m.ID
	cloned.Timestamp = time.Now().UnixMilli()
	if cloned.Usage.TotalTokens == 0 {
		cloned.Usage = model.Usage{}
	}
	return &cloned
}

func (p *FauxProvider) errorMessage(text string, m *ProviderModel) *model.AssistantMessage {
	return &model.AssistantMessage{
		Role:         "assistant",
		Content:      []model.ContentBlock{},
		API:          m.API,
		Provider:     m.Provider,
		Model:        m.ID,
		Usage:        model.Usage{},
		StopReason:   model.StopReasonError,
		ErrorMessage: text,
		Timestamp:    time.Now().UnixMilli(),
	}
}

// Usage estimation

func estimateTokens(text string) int {
	return int(math.Ceil(float64(len(text)) / 4.0))
}

func contentToText(blocks []model.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case model.ContentTypeText:
			parts = append(parts, b.Text)
		case model.ContentTypeImage:
			parts = append(parts, fmt.Sprintf("[image:%s:%d]", b.MimeType, len(b.Data)))
		case model.ContentTypeThinking:
			parts = append(parts, b.Thinking)
		case model.ContentTypeToolCall:
			parts = append(parts, fmt.Sprintf("%s:%s", b.Name, string(b.Arguments)))
		}
	}
	return strings.Join(parts, "\n")
}

func messageRole(data json.RawMessage) string {
	var m struct {
		Role string `json:"role"`
	}
	json.Unmarshal(data, &m)
	return m.Role
}

func serializeContext(c *Context) string {
	var parts []string
	if c.SystemPrompt != "" {
		parts = append(parts, "system:"+c.SystemPrompt)
	}
	for _, raw := range c.Messages {
		role := messageRole(raw)
		text, _ := json.Marshal(raw)
		parts = append(parts, role+":"+string(text))
	}
	if len(c.Tools) > 0 {
		toolJSON, _ := json.Marshal(c.Tools)
		parts = append(parts, "tools:"+string(toolJSON))
	}
	return strings.Join(parts, "\n\n")
}

func commonPrefixLen(a, b string) int {
	l := len(a)
	if len(b) < l {
		l = len(b)
	}
	i := 0
	for i < l && a[i] == b[i] {
		i++
	}
	return i
}

func (p *FauxProvider) withUsageEstimate(msg *model.AssistantMessage, c *Context, opts *StreamOptions) *model.AssistantMessage {
	promptText := serializeContext(c)
	promptTokens := estimateTokens(promptText)
	outputTokens := estimateTokens(contentToText(msg.Content))

	input := promptTokens
	var cacheRead, cacheWrite int

	if opts != nil && opts.SessionID != "" && opts.CacheRetention != "none" {
		p.mu.Lock()
		prev, exists := p.promptCache[opts.SessionID]
		if exists {
			cached := commonPrefixLen(prev, promptText)
			cacheRead = estimateTokens(prev[:cached])
			cacheWrite = estimateTokens(promptText[cached:])
			input = promptTokens - cacheRead
			if input < 0 {
				input = 0
			}
		} else {
			cacheWrite = promptTokens
		}
		p.promptCache[opts.SessionID] = promptText
		p.mu.Unlock()
	}

	msg.Usage = model.Usage{
		Input:       input,
		Output:      outputTokens,
		CacheRead:   cacheRead,
		CacheWrite:  cacheWrite,
		TotalTokens: input + outputTokens + cacheRead + cacheWrite,
		Cost:        model.UsageCost{},
	}
	return msg
}

// Stream with token-sized deltas

func splitByTokenSize(text string, minChars, maxChars int) []string {
	if len(text) == 0 {
		return []string{""}
	}
	var chunks []string
	i := 0
	for i < len(text) {
		size := minChars + randInt(maxChars-minChars+1)
		end := i + size
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
		i = end
	}
	if len(chunks) == 0 {
		return []string{""}
	}
	return chunks
}

var randState uint32 = 42

func randInt(max int) int {
	randState = randState*1103515245 + 12345
	return int(randState/65536) % max
}

func (p *FauxProvider) scheduleChunk(ctx context.Context, chunk string) error {
	if p.tokensPerSec <= 0 {
		return nil
	}
	delayMs := (float64(estimateTokens(chunk)) / p.tokensPerSec) * 1000
	timer := time.NewTimer(time.Duration(delayMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (p *FauxProvider) streamWithDeltas(ctx context.Context, ch chan<- model.StreamEvent, msg *model.AssistantMessage) {
	minChars := p.minTokenSize * 4
	maxChars := p.maxTokenSize * 4

	partial := &model.AssistantMessage{
		Role:       msg.Role,
		Content:    []model.ContentBlock{},
		API:        msg.API,
		Provider:   msg.Provider,
		Model:      msg.Model,
		Usage:      msg.Usage,
		StopReason: msg.StopReason,
		Timestamp:  msg.Timestamp,
	}

	if ctx.Err() != nil {
		aborted := p.abortMessage(partial)
		ch <- model.NewErrorEvent(model.StopReasonAborted, aborted)
		return
	}

	ch <- model.NewStartEvent(partial)

	for idx, block := range msg.Content {
		switch block.Type {
		case model.ContentTypeThinking:
			partial.Content = append(partial.Content, model.NewThinkingContent(""))
			ch <- model.NewThinkingStartEvent(idx, cp(partial))
			for _, chunk := range splitByTokenSize(block.Thinking, minChars, maxChars) {
				if err := p.scheduleChunk(ctx, chunk); err != nil {
					aborted := p.abortMessage(partial)
					ch <- model.NewErrorEvent(model.StopReasonAborted, aborted)
					return
				}
				partial.Content[len(partial.Content)-1].Thinking += chunk
				ch <- model.NewThinkingDeltaEvent(idx, chunk, cp(partial))
			}
			ch <- model.NewThinkingEndEvent(idx, block.Thinking, cp(partial))

		case model.ContentTypeText:
			partial.Content = append(partial.Content, model.NewTextContent(""))
			ch <- model.NewTextStartEvent(idx, cp(partial))
			for _, chunk := range splitByTokenSize(block.Text, minChars, maxChars) {
				if err := p.scheduleChunk(ctx, chunk); err != nil {
					aborted := p.abortMessage(partial)
					ch <- model.NewErrorEvent(model.StopReasonAborted, aborted)
					return
				}
				partial.Content[len(partial.Content)-1].Text += chunk
				ch <- model.NewTextDeltaEvent(idx, chunk, cp(partial))
			}
			ch <- model.NewTextEndEvent(idx, block.Text, cp(partial))

		case model.ContentTypeToolCall:
			partial.Content = append(partial.Content, model.NewToolCallContent(block.ID, block.Name, nil))
			ch <- model.NewToolCallStartEvent(idx, cp(partial))
			argsJSON := string(block.Arguments)
			for _, chunk := range splitByTokenSize(argsJSON, minChars, maxChars) {
				if err := p.scheduleChunk(ctx, chunk); err != nil {
					aborted := p.abortMessage(partial)
					ch <- model.NewErrorEvent(model.StopReasonAborted, aborted)
					return
				}
				ch <- model.NewToolCallDeltaEvent(idx, chunk, cp(partial))
			}
			partial.Content[len(partial.Content)-1].Arguments = block.Arguments
			tc := block
			ch <- model.NewToolCallEndEvent(idx, &tc, cp(partial))
		}
	}

	if msg.StopReason == model.StopReasonError || msg.StopReason == model.StopReasonAborted {
		ch <- model.NewErrorEvent(msg.StopReason, msg)
		return
	}

	ch <- model.NewDoneEvent(msg.StopReason, msg)
}

func (p *FauxProvider) abortMessage(partial *model.AssistantMessage) *model.AssistantMessage {
	return &model.AssistantMessage{
		Role:         "assistant",
		Content:      append([]model.ContentBlock{}, partial.Content...),
		API:          partial.API,
		Provider:     partial.Provider,
		Model:        partial.Model,
		Usage:        partial.Usage,
		StopReason:   model.StopReasonAborted,
		ErrorMessage: "Request was aborted",
		Timestamp:    time.Now().UnixMilli(),
	}
}

func cp(m *model.AssistantMessage) *model.AssistantMessage {
	c := *m
	c.Content = append([]model.ContentBlock{}, m.Content...)
	return &c
}

// Helper constructors

// FauxText creates a text content block.
func FauxText(text string) model.ContentBlock {
	return model.NewTextContent(text)
}

// FauxThinking creates a thinking content block.
func FauxThinking(thinking string) model.ContentBlock {
	return model.NewThinkingContent(thinking)
}

// FauxToolCall creates a toolCall content block.
func FauxToolCall(id, name string, args json.RawMessage) model.ContentBlock {
	return model.NewToolCallContent(id, name, args)
}

// FauxAssistantMessage creates a scripted assistant message for the response queue.
func FauxAssistantMessage(content []model.ContentBlock, stopReason model.StopReason) *model.AssistantMessage {
	if stopReason == "" {
		stopReason = model.StopReasonStop
	}
	return &model.AssistantMessage{
		Role:       "assistant",
		Content:    content,
		API:        defaultFauxAPI,
		Provider:   defaultFauxProvider,
		Model:      defaultFauxModelID,
		Usage:      model.Usage{},
		StopReason: stopReason,
		Timestamp:  time.Now().UnixMilli(),
	}
}
