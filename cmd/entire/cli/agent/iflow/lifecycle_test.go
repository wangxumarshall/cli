package iflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

func TestParseSessionStart(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected agent.EventType
		wantErr  bool
	}{
		{
			name: "valid session start",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "SessionStart",
				"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
				"source":          "startup",
				"model":           "glm-5",
			},
			expected: agent.SessionStart,
			wantErr:  false,
		},
		{
			name: "session start without model",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "SessionStart",
			},
			expected: agent.SessionStart,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.input)
			stdin := strings.NewReader(string(data))

			event, err := ag.ParseHookEvent(context.Background(), HookNameSessionStart, stdin)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if event.Type != tt.expected {
				t.Errorf("Expected event type %v, got %v", tt.expected, event.Type)
			}

			if event.SessionID != tt.input["session_id"] {
				t.Errorf("Expected session ID %q, got %q", tt.input["session_id"], event.SessionID)
			}

			if model, ok := tt.input["model"].(string); ok && event.Model != model {
				t.Errorf("Expected model %q, got %q", model, event.Model)
			}
		})
	}
}

func TestParseUserPromptSubmit(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/home/user/project",
		"hook_event_name": "UserPromptSubmit",
		"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
		"prompt":          "Write a function to calculate fibonacci numbers",
	}

	data, _ := json.Marshal(input)
	stdin := strings.NewReader(string(data))

	event, err := ag.ParseHookEvent(context.Background(), HookNameUserPromptSubmit, stdin)
	if err != nil {
		t.Fatalf("ParseHookEvent failed: %v", err)
	}

	if event.Type != agent.TurnStart {
		t.Errorf("Expected event type TurnStart, got %v", event.Type)
	}

	if event.Prompt != input["prompt"] {
		t.Errorf("Expected prompt %q, got %q", input["prompt"], event.Prompt)
	}
}

func TestParseStop(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedType  agent.EventType
		expectedDur   int64
		expectedTurns int
	}{
		{
			name: "stop with metrics",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "Stop",
				"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
				"duration_ms":     5000,
				"turn_count":      3,
			},
			expectedType:  agent.TurnEnd,
			expectedDur:   5000,
			expectedTurns: 3,
		},
		{
			name: "stop without metrics",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "Stop",
			},
			expectedType:  agent.TurnEnd,
			expectedDur:   0,
			expectedTurns: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.input)
			stdin := strings.NewReader(string(data))

			event, err := ag.ParseHookEvent(context.Background(), HookNameStop, stdin)
			if err != nil {
				t.Fatalf("ParseHookEvent failed: %v", err)
			}

			if event.Type != tt.expectedType {
				t.Errorf("Expected event type %v, got %v", tt.expectedType, event.Type)
			}

			if event.DurationMs != tt.expectedDur {
				t.Errorf("Expected duration %d, got %d", tt.expectedDur, event.DurationMs)
			}

			if event.TurnCount != tt.expectedTurns {
				t.Errorf("Expected turn count %d, got %d", tt.expectedTurns, event.TurnCount)
			}
		})
	}
}

func TestParseSessionEnd(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/home/user/project",
		"hook_event_name": "SessionEnd",
		"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
	}

	data, _ := json.Marshal(input)
	stdin := strings.NewReader(string(data))

	event, err := ag.ParseHookEvent(context.Background(), HookNameSessionEnd, stdin)
	if err != nil {
		t.Fatalf("ParseHookEvent failed: %v", err)
	}

	if event.Type != agent.SessionEnd {
		t.Errorf("Expected event type SessionEnd, got %v", event.Type)
	}

	if event.SessionID != input["session_id"] {
		t.Errorf("Expected session ID %q, got %q", input["session_id"], event.SessionID)
	}
}

func TestParseSubagentStop(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/home/user/project",
		"hook_event_name": "SubagentStop",
		"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
		"subagent_id":     "subagent-123",
		"duration_ms":     2000,
	}

	data, _ := json.Marshal(input)
	stdin := strings.NewReader(string(data))

	event, err := ag.ParseHookEvent(context.Background(), HookNameSubagentStop, stdin)
	if err != nil {
		t.Fatalf("ParseHookEvent failed: %v", err)
	}

	if event.Type != agent.SubagentEnd {
		t.Errorf("Expected event type SubagentEnd, got %v", event.Type)
	}

	if event.SubagentID != input["subagent_id"] {
		t.Errorf("Expected subagent ID %q, got %q", input["subagent_id"], event.SubagentID)
	}

	if event.DurationMs != int64(input["duration_ms"].(int)) {
		t.Errorf("Expected duration %d, got %d", input["duration_ms"], event.DurationMs)
	}
}

func TestParseHookEvent_UnknownHook(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	// Unknown hooks should return nil, nil
	event, err := ag.ParseHookEvent(context.Background(), "unknown-hook", strings.NewReader("{}"))
	if err != nil {
		t.Errorf("Expected no error for unknown hook, got %v", err)
	}
	if event != nil {
		t.Errorf("Expected nil event for unknown hook, got %v", event)
	}
}

func TestParseHookEvent_NonLifecycleHooks(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	// SetUpEnvironment and Notification should return nil, nil
	nonLifecycleHooks := []string{HookNameSetUpEnvironment, HookNameNotification}

	for _, hookName := range nonLifecycleHooks {
		t.Run(hookName, func(t *testing.T) {
			event, err := ag.ParseHookEvent(context.Background(), hookName, strings.NewReader("{}"))
			if err != nil {
				t.Errorf("Expected no error for %s, got %v", hookName, err)
			}
			if event != nil {
				t.Errorf("Expected nil event for %s, got %v", hookName, event)
			}
		})
	}
}

func TestWriteHookResponse(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	// This test just verifies the function doesn't panic
	// In a real test, we'd capture stdout
	err := ag.WriteHookResponse("Test message")
	if err != nil {
		t.Errorf("WriteHookResponse failed: %v", err)
	}
}

func TestEventTimestamp(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/home/user/project",
		"hook_event_name": "SessionStart",
	}

	data, _ := json.Marshal(input)
	stdin := strings.NewReader(string(data))

	before := time.Now()
	event, err := ag.ParseHookEvent(context.Background(), HookNameSessionStart, stdin)
	after := time.Now()

	if err != nil {
		t.Fatalf("ParseHookEvent failed: %v", err)
	}

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Error("Event timestamp is not within expected range")
	}
}

func TestParsePreToolUse(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "pre-tool-use with file_path",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "PreToolUse",
				"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
				"tool_name":       "write_file",
				"tool_aliases":    []string{"write", "create"},
				"tool_input": map[string]interface{}{
					"file_path": "/home/user/project/test.go",
					"content":   "package main",
				},
			},
		},
		{
			name: "pre-tool-use with edit tool",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "PreToolUse",
				"tool_name":       "replace",
				"tool_aliases":    []string{"Edit", "edit"},
				"tool_input": map[string]interface{}{
					"file_path":  "/home/user/project/main.go",
					"old_string": "old",
					"new_string": "new",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.input)
			stdin := strings.NewReader(string(data))

			// PreToolUse currently returns nil (no lifecycle event)
			event, err := ag.ParseHookEvent(context.Background(), HookNamePreToolUse, stdin)
			if err != nil {
				t.Errorf("ParseHookEvent failed: %v", err)
			}
			// PreToolUse doesn't generate a lifecycle event in current implementation
			if event != nil {
				t.Errorf("Expected nil event for PreToolUse, got %v", event)
			}
		})
	}
}

func TestParsePostToolUse(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "post-tool-use with response",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "PostToolUse",
				"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
				"tool_name":       "write_file",
				"tool_aliases":    []string{"write", "create"},
				"tool_input": map[string]interface{}{
					"file_path": "/home/user/project/test.go",
					"content":   "package main",
				},
				"tool_response": map[string]interface{}{
					"result": map[string]interface{}{
						"llmContent": "File written successfully",
					},
				},
			},
		},
		{
			name: "post-tool-use without response",
			input: map[string]interface{}{
				"session_id":      "test-session",
				"cwd":             "/home/user/project",
				"hook_event_name": "PostToolUse",
				"tool_name":       "read_file",
				"tool_input": map[string]interface{}{
					"file_path": "/home/user/project/main.go",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.input)
			stdin := strings.NewReader(string(data))

			// PostToolUse currently returns nil (no lifecycle event)
			event, err := ag.ParseHookEvent(context.Background(), HookNamePostToolUse, stdin)
			if err != nil {
				t.Errorf("ParseHookEvent failed: %v", err)
			}
			// PostToolUse doesn't generate a lifecycle event in current implementation
			if event != nil {
				t.Errorf("Expected nil event for PostToolUse, got %v", event)
			}
		})
	}
}

func TestParseSetUpEnvironment(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/home/user/project",
		"hook_event_name": "SetUpEnvironment",
		"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
	}

	data, _ := json.Marshal(input)
	stdin := strings.NewReader(string(data))

	// SetUpEnvironment returns nil (no lifecycle event)
	event, err := ag.ParseHookEvent(context.Background(), HookNameSetUpEnvironment, stdin)
	if err != nil {
		t.Errorf("ParseHookEvent failed: %v", err)
	}
	if event != nil {
		t.Errorf("Expected nil event for SetUpEnvironment, got %v", event)
	}
}

func TestParseNotification(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	input := map[string]interface{}{
		"session_id":      "test-session",
		"cwd":             "/home/user/project",
		"hook_event_name": "Notification",
		"transcript_path": "/home/user/.iflow/projects/test-session.jsonl",
		"message":         "Permission request for file access",
	}

	data, _ := json.Marshal(input)
	stdin := strings.NewReader(string(data))

	// Notification returns nil (no lifecycle event)
	event, err := ag.ParseHookEvent(context.Background(), HookNameNotification, stdin)
	if err != nil {
		t.Errorf("ParseHookEvent failed: %v", err)
	}
	if event != nil {
		t.Errorf("Expected nil event for Notification, got %v", event)
	}
}
