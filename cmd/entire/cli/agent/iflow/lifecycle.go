package iflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

// Ensure IFlowCLIAgent implements required interfaces
var (
	_ agent.TranscriptAnalyzer = (*IFlowCLIAgent)(nil)
	_ agent.HookResponseWriter = (*IFlowCLIAgent)(nil)
	_ agent.TokenCalculator    = (*IFlowCLIAgent)(nil)
)

// WriteHookResponse outputs a JSON hook response to stdout.
// iFlow CLI can read this JSON and display messages to the user.
func (i *IFlowCLIAgent) WriteHookResponse(message string) error {
	resp := struct {
		SystemMessage string `json:"systemMessage,omitempty"`
	}{SystemMessage: message}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		return fmt.Errorf("failed to encode hook response: %w", err)
	}
	return nil
}

// ParseHookEvent translates an iFlow CLI hook into a normalized lifecycle Event.
// Returns nil if the hook has no lifecycle significance.
func (i *IFlowCLIAgent) ParseHookEvent(_ context.Context, hookName string, stdin io.Reader) (*agent.Event, error) {
	switch hookName {
	case HookNameSessionStart:
		return i.parseSessionStart(stdin)
	case HookNameUserPromptSubmit:
		return i.parseTurnStart(stdin)
	case HookNamePreToolUse:
		return i.parsePreToolUse(stdin)
	case HookNamePostToolUse:
		return i.parsePostToolUse(stdin)
	case HookNameStop:
		return i.parseStop(stdin)
	case HookNameSessionEnd:
		return i.parseSessionEnd(stdin)
	case HookNameSubagentStop:
		return i.parseSubagentStop(stdin)
	case HookNameSetUpEnvironment, HookNameNotification:
		// These hooks don't have lifecycle significance for Entire
		return nil, nil
	default:
		return nil, nil
	}
}

// --- Internal hook parsing functions ---

func (i *IFlowCLIAgent) parseSessionStart(stdin io.Reader) (*agent.Event, error) {
	var input SessionStartInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode session start input: %w", err)
	}

	event := &agent.Event{
		Type:       agent.SessionStart,
		SessionID:  input.SessionID,
		SessionRef: input.TranscriptPath,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}

	if input.Model != "" {
		event.Model = input.Model
	}

	if input.Source != "" {
		event.Metadata["session_source"] = input.Source
	}

	return event, nil
}

func (i *IFlowCLIAgent) parseTurnStart(stdin io.Reader) (*agent.Event, error) {
	var input UserPromptSubmitInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode user prompt submit input: %w", err)
	}

	return &agent.Event{
		Type:       agent.TurnStart,
		SessionID:  input.SessionID,
		SessionRef: input.TranscriptPath,
		Prompt:     input.Prompt,
		Timestamp:  time.Now(),
	}, nil
}

func (i *IFlowCLIAgent) parsePreToolUse(stdin io.Reader) (*agent.Event, error) {
	var input ToolHookInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode pre-tool-use input: %w", err)
	}

	// Check if this is a subagent start (iFlow doesn't have explicit subagent concept,
	// but we can detect certain patterns if needed)
	// For now, we don't generate lifecycle events for PreToolUse
	// unless it's a special tool that indicates subagent behavior

	return nil, nil
}

func (i *IFlowCLIAgent) parsePostToolUse(stdin io.Reader) (*agent.Event, error) {
	var input ToolHookInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode post-tool-use input: %w", err)
	}

	// Similar to PreToolUse, we don't generate lifecycle events for PostToolUse
	// unless special handling is needed

	return nil, nil
}

func (i *IFlowCLIAgent) parseStop(stdin io.Reader) (*agent.Event, error) {
	var input StopInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode stop input: %w", err)
	}

	event := &agent.Event{
		Type:       agent.TurnEnd,
		SessionID:  input.SessionID,
		SessionRef: input.TranscriptPath,
		Timestamp:  time.Now(),
	}

	if input.DurationMs > 0 {
		event.DurationMs = input.DurationMs
	}
	if input.TurnCount > 0 {
		event.TurnCount = input.TurnCount
	}

	return event, nil
}

func (i *IFlowCLIAgent) parseSessionEnd(stdin io.Reader) (*agent.Event, error) {
	var input BaseHookInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode session end input: %w", err)
	}

	return &agent.Event{
		Type:       agent.SessionEnd,
		SessionID:  input.SessionID,
		SessionRef: input.TranscriptPath,
		Timestamp:  time.Now(),
	}, nil
}

func (i *IFlowCLIAgent) parseSubagentStop(stdin io.Reader) (*agent.Event, error) {
	var input SubagentStopInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode subagent stop input: %w", err)
	}

	event := &agent.Event{
		Type:       agent.SubagentEnd,
		SessionID:  input.SessionID,
		SessionRef: input.TranscriptPath,
		Timestamp:  time.Now(),
	}

	if input.SubagentID != "" {
		event.SubagentID = input.SubagentID
	}
	if input.DurationMs > 0 {
		event.DurationMs = input.DurationMs
	}

	return event, nil
}
