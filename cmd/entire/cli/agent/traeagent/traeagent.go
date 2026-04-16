// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/agent/types"
	"github.com/entireio/cli/cmd/entire/cli/paths"
)

//nolint:gochecknoinits // Agent self-registration is the intended pattern
func init() {
	agent.Register(agent.AgentNameTraeAgent, NewTraeAgent)
}

// 编译时接口断言
var _ agent.Agent = (*TraeAgent)(nil)
var _ agent.HookSupport = (*TraeAgent)(nil)
var _ agent.TranscriptAnalyzer = (*TraeAgent)(nil)

// TraeAgent implements the Agent interface for Trae Agent.
type TraeAgent struct{}

// NewTraeAgent creates a new Trae Agent instance.
func NewTraeAgent() agent.Agent {
	return &TraeAgent{}
}

// Name returns the agent registry key.
func (t *TraeAgent) Name() types.AgentName {
	return agent.AgentNameTraeAgent
}

// Type returns the agent type identifier.
func (t *TraeAgent) Type() types.AgentType {
	return agent.AgentTypeTraeAgent
}

// Description returns a human-readable description.
func (t *TraeAgent) Description() string {
	return "Trae Agent - ByteDance's LLM-based software engineering agent"
}

// IsPreview returns whether the agent integration is in preview.
func (t *TraeAgent) IsPreview() bool {
	return true
}

// DetectPresence checks if Trae Agent is configured in the repository.
func (t *TraeAgent) DetectPresence(ctx context.Context) (bool, error) {
	// Get repo root to check for .trae directory
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		// Not in a git repo, fall back to CWD-relative check
		repoRoot = "."
	}

	// Check for .trae directory
	traeDir := filepath.Join(repoRoot, ".trae")
	if _, err := os.Stat(traeDir); err == nil {
		return true, nil
	}
	// Check for .trae/settings.json or trae_config.yaml
	settingsFile := filepath.Join(repoRoot, ".trae", "settings.json")
	if _, err := os.Stat(settingsFile); err == nil {
		return true, nil
	}
	configFile := filepath.Join(repoRoot, "trae_config.yaml")
	if _, err := os.Stat(configFile); err == nil {
		return true, nil
	}
	return false, nil
}

// GetHookConfigPath returns the path to Trae's hook config file.
func (t *TraeAgent) GetHookConfigPath() string {
	return ".trae/settings.json"
}

// SupportsHooks returns true as Trae Agent supports lifecycle hooks.
func (t *TraeAgent) SupportsHooks() bool {
	return true
}

// ParseHookInput parses Trae Agent hook input from stdin.
func (t *TraeAgent) ParseHookInput(hookType agent.HookType, reader io.Reader) (*agent.HookInput, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	if len(data) == 0 {
		return nil, errors.New("empty input")
	}

	input := &agent.HookInput{
		HookType:  hookType,
		Timestamp: time.Now(),
		RawData:   make(map[string]interface{}),
	}

	// Parse based on hook type
	switch hookType {
	case agent.HookSessionStart:
		var raw sessionStartRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse session start: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath

	case agent.HookSessionEnd:
		var raw sessionEndRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse session end: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath

	case agent.HookPreToolUse:
		var raw preToolUseRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse pre-tool input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath
		input.ToolName = raw.ToolName
		input.ToolUseID = raw.ToolUseID
		input.ToolInput = data

	case agent.HookPostToolUse:
		var raw postToolUseRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse post-tool input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath
		input.ToolName = raw.ToolName
		input.ToolUseID = raw.ToolUseID
		input.ToolInput = data
		input.ToolResponse = data

	case agent.HookBeforeModel:
		var raw preModelRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse pre-model input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath
		input.RawData["model_name"] = raw.ModelName
		input.RawData["prompt"] = raw.Prompt

	case agent.HookAfterModel:
		var raw postModelRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse post-model input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath
		input.RawData["model_name"] = raw.ModelName
		input.RawData["response"] = raw.Response
		input.RawData["token_usage"] = raw.TokenUsage

	case agent.HookBeforeAgent:
		var raw beforeAgentRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse before-agent input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath
		input.RawData["prompt"] = raw.Prompt

	case agent.HookAfterAgent:
		var raw afterAgentRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse after-agent input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath

	case agent.HookPreCompress:
		var raw preCompressRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse pre-compress input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath
		input.RawData["context"] = raw.Context

	case agent.HookNotification:
		var raw notificationRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse notification input: %w", err)
		}
		input.SessionID = raw.SessionID
		input.SessionRef = raw.TrajectoryPath
		input.RawData["notification_type"] = raw.NotificationType
		input.RawData["message"] = raw.Message

	// Supported hooks that don't require special parsing
	case agent.HookUserPromptSubmit,
		agent.HookStop,
		agent.HookBeforeToolSelection,
		agent.HookPreTool,
		agent.HookAfterTool:
		// These hooks are supported but don't need special parsing
		// The RawData will contain the full input

	default:
		// Unknown hook type - still store raw data
	}

	return input, nil
}

// GetSessionID extracts the session ID from hook input.
func (t *TraeAgent) GetSessionID(input *agent.HookInput) string {
	return input.SessionID
}

// ResolveSessionFile returns the path to a Trae Agent session file.
func (t *TraeAgent) ResolveSessionFile(sessionDir, agentSessionID string) string {
	return filepath.Join(sessionDir, agentSessionID+"_trajectory.json")
}

// ProtectedDirs returns directories that Trae Agent uses for config/state.
func (t *TraeAgent) ProtectedDirs() []string { return []string{".trae"} }

// GetSessionDir returns the directory where Trae Agent stores session transcripts.
func (t *TraeAgent) GetSessionDir(repoPath string) (string, error) {
	// Check for test environment override
	if override := os.Getenv("ENTIRE_TEST_TRAE_PROJECT_DIR"); override != "" {
		return override, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".trae", "projects", filepath.Base(repoPath)), nil
}

// ReadSession reads a session from Trae Agent's storage (JSON trajectory file).
func (t *TraeAgent) ReadSession(input *agent.HookInput) (*agent.AgentSession, error) {
	if input.SessionRef == "" {
		return nil, errors.New("session reference (trajectory path) is required")
	}

	// Read the raw JSON trajectory file
	data, err := os.ReadFile(input.SessionRef)
	if err != nil {
		return nil, fmt.Errorf("failed to read trajectory: %w", err)
	}

	// Parse to extract computed fields
	modifiedFiles, err := ExtractModifiedFiles(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract modified files: %w", err)
	}

	return &agent.AgentSession{
		SessionID:     input.SessionID,
		AgentName:     t.Name(),
		SessionRef:    input.SessionRef,
		StartTime:     time.Now(),
		NativeData:    data,
		ModifiedFiles: modifiedFiles,
	}, nil
}

// WriteSession writes a session to Trae Agent's storage (JSON trajectory file).
func (t *TraeAgent) WriteSession(_ context.Context, session *agent.AgentSession) error {
	if session == nil {
		return errors.New("session is nil")
	}

	// Verify this session belongs to Trae Agent
	if session.AgentName != "" && session.AgentName != t.Name() {
		return fmt.Errorf("session belongs to agent %q, not %q", session.AgentName, t.Name())
	}

	if session.SessionRef == "" {
		return errors.New("session reference (trajectory path) is required")
	}

	if len(session.NativeData) == 0 {
		return errors.New("session has no native data to write")
	}

	// Write the raw JSON data
	if err := os.WriteFile(session.SessionRef, session.NativeData, 0o600); err != nil {
		return fmt.Errorf("failed to write trajectory: %w", err)
	}

	return nil
}

// FormatResumeCommand returns the command to resume a Trae Agent session.
func (t *TraeAgent) FormatResumeCommand(sessionID string) string {
	return "trae-cli interactive --resume " + sessionID
}

// ReadTranscript reads the raw JSON trajectory bytes for a session.
func (t *TraeAgent) ReadTranscript(sessionRef string) ([]byte, error) {
	data, err := os.ReadFile(sessionRef) //nolint:gosec // Path comes from agent hook input
	if err != nil {
		return nil, fmt.Errorf("failed to read trajectory: %w", err)
	}
	return data, nil
}

// ExtractModifiedFiles extracts modified files from Trae Agent trajectory data.
func ExtractModifiedFiles(data []byte) ([]string, error) {
	return extractModifiedFiles(data)
}
