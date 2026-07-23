// Package ai is the public API surface for pi-go's AI core, equivalent to
// @earendil-works/pi-ai in the TypeScript reference implementation.
//
// It re-exports all types and functions needed to work with AI providers,
// models, messages, streaming, and credentials.
//
// Usage:
//
//	import ai "github.com/DROWNING2003/pi-go/packages/ai"
//
//	// Work with messages
//	msg := ai.NewUserMessage("hello")
//
//	// Use the provider registry
//	reg := ai.NewRegistry(nil)
//	ai.RegisterBuiltins(reg)
//	m := reg.ResolveModel("deepseek/deepseek-chat")
//
//	// Stream from a provider
//	events := ai.StreamChatCompletion(ctx, client, model, context, opts)
package ai

import (
	"github.com/DROWNING2003/pi-go/packages/ai/model"
	"github.com/DROWNING2003/pi-go/packages/ai/protocol"
	"github.com/DROWNING2003/pi-go/packages/ai/provider"
)

// --- Model types ---

type (
	// ContentBlock represents any content block in a message.
	ContentBlock = model.ContentBlock
	// UserContent is polymorphic content (string or []ContentBlock).
	UserContent = model.UserContent
	// Usage represents token usage and cost.
	Usage = model.Usage
	// UsageCost represents cost breakdown.
	UsageCost = model.UsageCost
	// StopReason describes why the model stopped.
	StopReason = model.StopReason
	// UserMessage represents a user message.
	UserMessage = model.UserMessage
	// AssistantMessage represents a model response.
	AssistantMessage = model.AssistantMessage
	// ToolResultMessage represents a tool execution result.
	ToolResultMessage = model.ToolResultMessage
	// StreamEvent represents a provider stream event.
	StreamEvent = model.StreamEvent
	// UnifiedModel is the unified model descriptor.
	UnifiedModel = model.UnifiedModel
	// Context is the conversation context.
	Context = model.Context
	// ToolDef describes a tool.
	ToolDef = model.ToolDef
	// ModelCost holds pricing.
	ModelCost = model.ModelCost
	// ThinkingLevel controls reasoning effort.
	ThinkingLevel = model.ThinkingLevel
	// CacheRetention controls cache policy.
	CacheRetention = model.CacheRetention
)

// Content block types.
const (
	ContentTypeText     = model.ContentTypeText
	ContentTypeThinking = model.ContentTypeThinking
	ContentTypeImage    = model.ContentTypeImage
	ContentTypeToolCall = model.ContentTypeToolCall
)

// Stop reason values.
const (
	StopReasonStop    = model.StopReasonStop
	StopReasonLength  = model.StopReasonLength
	StopReasonToolUse = model.StopReasonToolUse
	StopReasonError   = model.StopReasonError
	StopReasonAborted = model.StopReasonAborted
)

// Stream event types.
const (
	StreamEventStart         = model.StreamEventStart
	StreamEventTextDelta     = model.StreamEventTextDelta
	StreamEventTextEnd       = model.StreamEventTextEnd
	StreamEventThinkingStart = model.StreamEventThinkingStart
	StreamEventThinkingDelta = model.StreamEventThinkingDelta
	StreamEventDone          = model.StreamEventDone
	StreamEventError         = model.StreamEventError
)

// Thinking levels.
const (
	ThinkingOff     = model.ThinkingOff
	ThinkingMinimal = model.ThinkingMinimal
	ThinkingLow     = model.ThinkingLow
	ThinkingMedium  = model.ThinkingMedium
	ThinkingHigh    = model.ThinkingHigh
)

// Cache retention values.
const (
	CacheNone  = model.CacheRetentionNone
	CacheShort = model.CacheRetentionShort
	CacheLong  = model.CacheRetentionLong
)

// --- Constructor functions ---

var (
	NewTextContent        = model.NewTextContent
	NewThinkingContent    = model.NewThinkingContent
	NewImageContent       = model.NewImageContent
	NewToolCallContent    = model.NewToolCallContent
	NewStartEvent         = model.NewStartEvent
	NewTextDeltaEvent     = model.NewTextDeltaEvent
	NewTextEndEvent       = model.NewTextEndEvent
	NewThinkingStartEvent = model.NewThinkingStartEvent
	NewThinkingDeltaEvent = model.NewThinkingDeltaEvent
	NewThinkingEndEvent   = model.NewThinkingEndEvent
	NewToolCallStartEvent = model.NewToolCallStartEvent
	NewToolCallDeltaEvent = model.NewToolCallDeltaEvent
	NewToolCallEndEvent   = model.NewToolCallEndEvent
	NewDoneEvent          = model.NewDoneEvent
	NewErrorEvent         = model.NewErrorEvent
)

// --- Provider types ---

type (
	// ProviderConfig describes a registered AI provider.
	ProviderConfig = provider.ProviderConfig
	// ModelConfig is a provider-specific model definition.
	ModelConfig = provider.ModelConfig
	// Registry holds all registered providers.
	Registry = provider.Registry
	// CredentialStore stores API credentials.
	CredentialStore = provider.CredentialStore
	// Credential represents a stored credential.
	Credential = provider.Credential
	// CompatConfig holds provider-specific compat settings.
	CompatConfig = provider.CompatConfig
	// StreamDispatcher dispatches provider streams.
	StreamDispatcher = provider.StreamDispatcher
)

// Registry constructors.
var (
	NewRegistry        = provider.NewRegistry
	RegisterBuiltins   = provider.RegisterBuiltins
	NewCredentialStore = provider.NewCredentialStore
	ResolveAPIKey      = provider.ResolveAPIKey
	DetectCompat       = provider.DetectCompat
)

// Faux provider types for testing.
type (
	FauxProvider        = provider.FauxProvider
	FauxResponseStep    = provider.FauxResponseStep
	FauxMessage         = provider.FauxMessage
	FauxResponseFactory = provider.FauxResponseFactory
	FauxProviderOption  = provider.FauxProviderOption
)

var (
	NewFauxProvider         = provider.NewFauxProvider
	FauxText                = provider.FauxText
	FauxThinking            = provider.FauxThinking
	FauxToolCall            = provider.FauxToolCall
	FauxAssistantMessage    = provider.FauxAssistantMessage
	WithFauxID              = provider.WithFauxID
	WithFauxModels          = provider.WithFauxModels
	WithFauxTokenSize       = provider.WithFauxTokenSize
	WithFauxTokensPerSecond = provider.WithFauxTokensPerSecond
)

// --- Protocol types ---

type (
	// HTTPClient is an HTTP client for provider APIs.
	HTTPClient = protocol.HTTPClient
	// SSEParser parses Server-Sent Events.
	SSEParser = protocol.SSEParser
	// JSONLinesParser parses JSONL/NDJSON streams.
	JSONLinesParser = protocol.JSONLinesParser
	// HTTPError represents an HTTP error response.
	HTTPError = protocol.HTTPError
)

// Protocol constructors and helpers.
var (
	NewHTTPClient           = protocol.NewHTTPClient
	NewSSEParser            = protocol.NewSSEParser
	NewJSONLinesParser      = protocol.NewJSONLinesParser
	SanitizeHeaders         = protocol.SanitizeHeaders
	StreamChatCompletion    = protocol.StreamChatCompletion
	StreamOpenAIResponses   = protocol.StreamOpenAIResponses
	StreamAnthropicMessages = protocol.StreamAnthropicMessages
	StreamGoogleGenerate    = protocol.StreamGoogleGenerate
)
