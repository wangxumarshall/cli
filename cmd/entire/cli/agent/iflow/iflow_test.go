package iflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

func TestIFlowCLIAgent_Name(t *testing.T) {
	ag := NewIFlowCLIAgent()
	if ag.Name() != agent.AgentNameIFlow {
		t.Errorf("Expected name %q, got %q", agent.AgentNameIFlow, ag.Name())
	}
}

func TestIFlowCLIAgent_Type(t *testing.T) {
	ag := NewIFlowCLIAgent()
	if ag.Type() != agent.AgentTypeIFlow {
		t.Errorf("Expected type %q, got %q", agent.AgentTypeIFlow, ag.Type())
	}
}

func TestIFlowCLIAgent_Description(t *testing.T) {
	ag := NewIFlowCLIAgent()
	expected := "iFlow CLI - Alibaba's AI coding assistant"
	if ag.Description() != expected {
		t.Errorf("Expected description %q, got %q", expected, ag.Description())
	}
}

func TestIFlowCLIAgent_IsPreview(t *testing.T) {
	ag := NewIFlowCLIAgent()
	if !ag.IsPreview() {
		t.Error("Expected IsPreview to return true")
	}
}

func TestIFlowCLIAgent_ProtectedDirs(t *testing.T) {
	ag := NewIFlowCLIAgent()
	dirs := ag.ProtectedDirs()
	if len(dirs) != 1 || dirs[0] != ".iflow" {
		t.Errorf("Expected protected dirs [.iflow], got %v", dirs)
	}
}

func TestIFlowCLIAgent_DetectPresence(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "detects .iflow directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.MkdirAll(filepath.Join(dir, ".iflow"), 0o755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			expected: true,
		},
		{
			name: "detects settings.json",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.MkdirAll(filepath.Join(dir, ".iflow"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".iflow", "settings.json"), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			expected: true,
		},
		{
			name: "no iFlow config",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			t.Chdir(dir)

			ag := NewIFlowCLIAgent()
			present, err := ag.DetectPresence(context.Background())
			if err != nil {
				t.Fatalf("DetectPresence failed: %v", err)
			}
			if present != tt.expected {
				t.Errorf("Expected DetectPresence=%v, got %v", tt.expected, present)
			}
		})
	}
}

func TestIFlowCLIAgent_FormatResumeCommand(t *testing.T) {
	ag := NewIFlowCLIAgent()
	sessionID := "test-session-123"
	cmd := ag.FormatResumeCommand(sessionID)
	expected := "iflow -r test-session-123"
	if cmd != expected {
		t.Errorf("Expected resume command %q, got %q", expected, cmd)
	}
}

func TestSanitizePathForIFlow(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/project", "-home-user-project"},
		{"my-project", "my-project"},
		{"my_project", "my-project"},
		{"my.project", "my-project"},
		{"my@project", "my-project"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizePathForIFlow(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePathForIFlow(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIFlowCLIAgent_ResolveSessionFile(t *testing.T) {
	ag := NewIFlowCLIAgent()
	sessionDir := "/home/user/.iflow/projects/my-project"
	sessionID := "test-session"

	path := ag.ResolveSessionFile(sessionDir, sessionID)
	expected := filepath.Join(sessionDir, "test-session.jsonl")
	if path != expected {
		t.Errorf("Expected session file path %q, got %q", expected, path)
	}
}

func TestIFlowCLIAgent_GetSessionID(t *testing.T) {
	ag := NewIFlowCLIAgent()
	input := &agent.HookInput{
		SessionID: "test-session-id",
	}

	sessionID := ag.GetSessionID(input)
	if sessionID != "test-session-id" {
		t.Errorf("Expected session ID %q, got %q", "test-session-id", sessionID)
	}
}

// Test interface compliance
func TestIFlowCLIAgent_ImplementsAgent(t *testing.T) {
	var _ agent.Agent = (*IFlowCLIAgent)(nil)
}

func TestIFlowCLIAgent_ImplementsHookSupport(t *testing.T) {
	var _ agent.HookSupport = (*IFlowCLIAgent)(nil)
}

func TestIFlowCLIAgent_ImplementsTranscriptAnalyzer(t *testing.T) {
	var _ agent.TranscriptAnalyzer = (*IFlowCLIAgent)(nil)
}

func TestIFlowCLIAgent_ImplementsHookResponseWriter(t *testing.T) {
	var _ agent.HookResponseWriter = (*IFlowCLIAgent)(nil)
}

// Ensure agent is registered
func TestIFlowCLIAgent_Registered(t *testing.T) {
	ag, err := agent.Get(agent.AgentNameIFlow)
	if err != nil {
		t.Fatalf("iFlow agent not registered: %v", err)
	}
	if ag == nil {
		t.Fatal("iFlow agent is nil")
	}
	if ag.Name() != agent.AgentNameIFlow {
		t.Errorf("Expected name %q, got %q", agent.AgentNameIFlow, ag.Name())
	}
}

// Test GetByAgentType
func TestGetByAgentType_IFlow(t *testing.T) {
	ag, err := agent.GetByAgentType(agent.AgentTypeIFlow)
	if err != nil {
		t.Fatalf("Failed to get iFlow agent by type: %v", err)
	}
	if ag.Type() != agent.AgentTypeIFlow {
		t.Errorf("Expected type %q, got %q", agent.AgentTypeIFlow, ag.Type())
	}
}

func TestIFlowCLIAgent_ImplementsTokenCalculator(t *testing.T) {
	var _ agent.TokenCalculator = (*IFlowCLIAgent)(nil)
}

func TestCalculateTokenUsage(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	tests := []struct {
		name        string
		transcript  string
		fromOffset  int
		expectUsage *agent.TokenUsage
		expectErr   bool
	}{
		{
			name:        "empty transcript",
			transcript:  "",
			fromOffset:  0,
			expectUsage: &agent.TokenUsage{},
			expectErr:   false,
		},
		{
			name: "transcript with tokens",
			transcript: `{"type":"user","timestamp":"2024-01-01T00:00:00Z","message":"hello"}
{"type":"assistant","timestamp":"2024-01-01T00:00:01Z","tokens":{"input":100,"output":50,"cached":10}}
{"type":"assistant","timestamp":"2024-01-01T00:00:02Z","tokens":{"input":200,"output":75,"cached":20}}`,
			fromOffset: 0,
			expectUsage: &agent.TokenUsage{
				InputTokens:     300,
				OutputTokens:    125,
				CacheReadTokens: 30,
				APICallCount:    2,
			},
			expectErr: false,
		},
		{
			name: "transcript with fromOffset",
			transcript: `{"type":"assistant","timestamp":"2024-01-01T00:00:01Z","tokens":{"input":100,"output":50}}
{"type":"assistant","timestamp":"2024-01-01T00:00:02Z","tokens":{"input":200,"output":75}}`,
			fromOffset: 1,
			expectUsage: &agent.TokenUsage{
				InputTokens:  200,
				OutputTokens: 75,
				APICallCount: 1,
			},
			expectErr: false,
		},
		{
			name: "transcript without tokens",
			transcript: `{"type":"user","timestamp":"2024-01-01T00:00:00Z","message":"hello"}
{"type":"assistant","timestamp":"2024-01-01T00:00:01Z","message":"hi there"}`,
			fromOffset:  0,
			expectUsage: &agent.TokenUsage{},
			expectErr:   false,
		},
		{
			name: "mixed message types",
			transcript: `{"type":"user","timestamp":"2024-01-01T00:00:00Z","message":"hello","tokens":{"input":50}}
{"type":"assistant","timestamp":"2024-01-01T00:00:01Z","tokens":{"input":100,"output":50}}
{"type":"ai","timestamp":"2024-01-01T00:00:02Z","tokens":{"input":200,"output":75}}`,
			fromOffset: 0,
			expectUsage: &agent.TokenUsage{
				InputTokens:  300,
				OutputTokens: 125,
				APICallCount: 2,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := ag.CalculateTokenUsage([]byte(tt.transcript), tt.fromOffset)
			if (err != nil) != tt.expectErr {
				t.Errorf("CalculateTokenUsage() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if usage.InputTokens != tt.expectUsage.InputTokens {
				t.Errorf("InputTokens = %d, want %d", usage.InputTokens, tt.expectUsage.InputTokens)
			}
			if usage.OutputTokens != tt.expectUsage.OutputTokens {
				t.Errorf("OutputTokens = %d, want %d", usage.OutputTokens, tt.expectUsage.OutputTokens)
			}
			if usage.CacheReadTokens != tt.expectUsage.CacheReadTokens {
				t.Errorf("CacheReadTokens = %d, want %d", usage.CacheReadTokens, tt.expectUsage.CacheReadTokens)
			}
			if usage.APICallCount != tt.expectUsage.APICallCount {
				t.Errorf("APICallCount = %d, want %d", usage.APICallCount, tt.expectUsage.APICallCount)
			}
		})
	}
}

func TestCalculateTokenUsage_InvalidJSON(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)

	// Invalid JSON should not cause panic, but return empty usage
	transcript := `{invalid json`
	usage, err := ag.CalculateTokenUsage([]byte(transcript), 0)

	// The current implementation should handle this gracefully
	// ParseTranscriptFromBytes skips malformed lines
	if err != nil {
		t.Errorf("Expected no error for malformed JSON, got %v", err)
	}
	if usage == nil {
		t.Error("Expected non-nil usage")
	}
}
