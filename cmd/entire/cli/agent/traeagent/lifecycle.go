// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import (
	"context"
	"io"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

// ParseHookEvent translates a Trae Agent hook into a normalized lifecycle Event.
// Returns nil if the hook has no lifecycle significance.
func (t *TraeAgent) ParseHookEvent(_ context.Context, hookName string, stdin io.Reader) (*agent.Event, error) {
	switch hookName {
	case HookNameSessionStart:
		return t.parseSessionStart(stdin)
	case HookNameBeforeAgent:
		return t.parseBeforeAgent(stdin)
	case HookNameAfterAgent:
		return t.parseAfterAgent(stdin)
	case HookNameSessionEnd:
		return t.parseSessionEnd(stdin)
	default:
		return nil, nil //nolint:nilnil // Unknown hooks have no lifecycle action
	}
}

func (t *TraeAgent) parseSessionStart(stdin io.Reader) (*agent.Event, error) {
	input, err := t.ParseHookInput(agent.HookSessionStart, stdin)
	if err != nil {
		return nil, err
	}
	return &agent.Event{
		Type:       agent.SessionStart,
		SessionID:  input.SessionID,
		SessionRef: input.SessionRef,
		Timestamp:  time.Now(),
	}, nil
}

func (t *TraeAgent) parseBeforeAgent(stdin io.Reader) (*agent.Event, error) {
	input, err := t.ParseHookInput(agent.HookBeforeAgent, stdin)
	if err != nil {
		return nil, err
	}
	prompt, _ := input.RawData["prompt"].(string) //nolint:errcheck // Type assertion result safely handled
	return &agent.Event{
		Type:       agent.TurnStart,
		SessionID:  input.SessionID,
		SessionRef: input.SessionRef,
		Prompt:     prompt,
		Timestamp:  time.Now(),
	}, nil
}

func (t *TraeAgent) parseAfterAgent(stdin io.Reader) (*agent.Event, error) {
	input, err := t.ParseHookInput(agent.HookAfterAgent, stdin)
	if err != nil {
		return nil, err
	}
	return &agent.Event{
		Type:       agent.TurnEnd,
		SessionID:  input.SessionID,
		SessionRef: input.SessionRef,
		Timestamp:  time.Now(),
	}, nil
}

func (t *TraeAgent) parseSessionEnd(stdin io.Reader) (*agent.Event, error) {
	input, err := t.ParseHookInput(agent.HookSessionEnd, stdin)
	if err != nil {
		return nil, err
	}
	return &agent.Event{
		Type:       agent.SessionEnd,
		SessionID:  input.SessionID,
		SessionRef: input.SessionRef,
		Timestamp:  time.Now(),
	}, nil
}
