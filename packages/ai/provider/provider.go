package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/DROWNING2003/pi-go/packages/ai/model"
)

type Context struct {
	SystemPrompt string
	Messages     []model.Message
}

type Stream interface {
	Events() <-chan model.StreamEvent
	Result() (model.AssistantMessage, error)
	Done() <-chan struct{}
}

type Provider interface {
	ID() model.ProviderID
	Models(context.Context) ([]model.Model, error)
	Stream(context.Context, model.Model, Context) Stream
}

type FauxResponse struct {
	Events []model.StreamEvent
	Result model.AssistantMessage
	Err    error
	Delay  time.Duration
}

type FauxProvider struct {
	mu        sync.Mutex
	responses []FauxResponse
	next      int
}

func NewFauxProvider(responses []FauxResponse) *FauxProvider {
	return &FauxProvider{responses: append([]FauxResponse(nil), responses...)}
}

func (p *FauxProvider) ID() model.ProviderID { return "faux" }

func (p *FauxProvider) Models(context.Context) ([]model.Model, error) {
	return []model.Model{{ID: "test", Name: "Faux Test", API: "faux", Provider: p.ID(), Input: []string{"text"}}}, nil
}

func (p *FauxProvider) Stream(ctx context.Context, _ model.Model, _ Context) Stream {
	p.mu.Lock()
	if p.next >= len(p.responses) {
		p.mu.Unlock()
		return newFauxStream(ctx, FauxResponse{Err: errors.New("faux provider has no scripted response")})
	}
	response := p.responses[p.next]
	p.next++
	p.mu.Unlock()
	return newFauxStream(ctx, response)
}

type fauxStream struct {
	events chan model.StreamEvent
	done   chan struct{}
	result model.AssistantMessage
	err    error
}

func newFauxStream(ctx context.Context, response FauxResponse) *fauxStream {
	stream := &fauxStream{
		events: make(chan model.StreamEvent),
		done:   make(chan struct{}),
	}
	go stream.run(ctx, response)
	return stream
}

func (s *fauxStream) Events() <-chan model.StreamEvent { return s.events }
func (s *fauxStream) Done() <-chan struct{}            { return s.done }

func (s *fauxStream) Result() (model.AssistantMessage, error) {
	<-s.done
	return s.result, s.err
}

func (s *fauxStream) run(ctx context.Context, response FauxResponse) {
	defer close(s.events)
	defer close(s.done)

	if response.Delay > 0 {
		timer := time.NewTimer(response.Delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			s.result = abortedResult(response.Result, ctx.Err())
			s.err = ctx.Err()
			return
		}
	}

	for _, event := range response.Events {
		select {
		case s.events <- event:
		case <-ctx.Done():
			s.result = abortedResult(response.Result, ctx.Err())
			s.err = ctx.Err()
			return
		}
	}

	if response.Err != nil {
		s.result = response.Result
		s.result.StopReason = model.StopReasonError
		s.result.ErrorMessage = response.Err.Error()
		s.err = response.Err
		return
	}
	s.result = response.Result
}

func abortedResult(result model.AssistantMessage, err error) model.AssistantMessage {
	result.StopReason = model.StopReasonAborted
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("stream aborted: %v", err)
	}
	return result
}
