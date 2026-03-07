// Package iflow implements the Agent interface for iFlow CLI.
// iFlow CLI is Alibaba's AI coding assistant with a hooks-based event system.
package iflow

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/agent/types"
	"github.com/entireio/cli/cmd/entire/cli/paths"
)

//nolint:gochecknoinits // Agent self-registration is the intended pattern
func init() {
	agent.Register(agent.AgentNameIFlow, NewIFlowCLIAgent)
}

// IFlowCLIAgent implements the Agent interface for iFlow CLI.
type IFlowCLIAgent struct{}

// NewIFlowCLIAgent creates a new iFlow CLI agent instance.
func NewIFlowCLIAgent() agent.Agent {
	return &IFlowCLIAgent{}
}

// Name returns the agent registry key.
func (i *IFlowCLIAgent) Name() types.AgentName {
	return agent.AgentNameIFlow
}

// Type returns the agent type identifier.
func (i *IFlowCLIAgent) Type() types.AgentType {
	return agent.AgentTypeIFlow
}

// Description returns a human-readable description.
func (i *IFlowCLIAgent) Description() string {
	return "iFlow CLI - Alibaba's AI coding assistant"
}

// IsPreview returns whether the agent integration is in preview.
func (i *IFlowCLIAgent) IsPreview() bool {
	return true
}

// DetectPresence checks if iFlow CLI is configured in the repository.
func (i *IFlowCLIAgent) DetectPresence(ctx context.Context) (bool, error) {
	// Get worktree root to check for .iflow directory
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		// Not in a git repo, fall back to CWD-relative check
		repoRoot = "."
	}

	// Check for .iflow directory
	iflowDir := filepath.Join(repoRoot, ".iflow")
	if _, err := os.Stat(iflowDir); err == nil {
		return true, nil
	}
	// Check for .iflow/settings.json
	settingsFile := filepath.Join(iflowDir, "settings.json")
	if _, err := os.Stat(settingsFile); err == nil {
		return true, nil
	}
	return false, nil
}

// ProtectedDirs returns directories that iFlow uses for config/state.
func (i *IFlowCLIAgent) ProtectedDirs() []string {
	return []string{".iflow"}
}

// GetSessionID extracts the session ID from hook input.
func (i *IFlowCLIAgent) GetSessionID(input *agent.HookInput) string {
	return input.SessionID
}

// GetSessionDir returns the directory where iFlow stores session transcripts.
func (i *IFlowCLIAgent) GetSessionDir(repoPath string) (string, error) {
	// Check for test environment override
	if override := os.Getenv("ENTIRE_TEST_IFLOW_PROJECT_DIR"); override != "" {
		return override, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	projectDir := SanitizePathForIFlow(repoPath)
	return filepath.Join(homeDir, ".iflow", "projects", projectDir), nil
}

// ResolveSessionFile returns the path to an iFlow session file.
// iFlow names files as <session_id>.jsonl
func (i *IFlowCLIAgent) ResolveSessionFile(sessionDir, agentSessionID string) string {
	return filepath.Join(sessionDir, agentSessionID+".jsonl")
}

// ReadSession reads a session from iFlow's storage (JSONL transcript file).
func (i *IFlowCLIAgent) ReadSession(input *agent.HookInput) (*agent.AgentSession, error) {
	if input.SessionRef == "" {
		return nil, errors.New("session reference (transcript path) is required")
	}

	// Read the raw JSONL file
	data, err := os.ReadFile(input.SessionRef)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	// Parse to extract computed fields
	lines, err := ParseTranscriptFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transcript: %w", err)
	}

	return &agent.AgentSession{
		SessionID:     input.SessionID,
		AgentName:     i.Name(),
		SessionRef:    input.SessionRef,
		StartTime:     time.Now(),
		NativeData:    data,
		ModifiedFiles: ExtractModifiedFiles(lines),
	}, nil
}

// WriteSession writes a session to iFlow's storage (JSONL transcript file).
func (i *IFlowCLIAgent) WriteSession(_ context.Context, session *agent.AgentSession) error {
	if session == nil {
		return errors.New("session is nil")
	}

	// Verify this session belongs to iFlow
	if session.AgentName != "" && session.AgentName != i.Name() {
		return fmt.Errorf("session belongs to agent %q, not %q", session.AgentName, i.Name())
	}

	if session.SessionRef == "" {
		return errors.New("session reference (transcript path) is required")
	}

	if len(session.NativeData) == 0 {
		return errors.New("session has no native data to write")
	}

	// Write the raw JSONL data
	if err := os.WriteFile(session.SessionRef, session.NativeData, 0o600); err != nil {
		return fmt.Errorf("failed to write transcript: %w", err)
	}

	return nil
}

// FormatResumeCommand returns the command to resume an iFlow CLI session.
func (i *IFlowCLIAgent) FormatResumeCommand(sessionID string) string {
	return "iflow -r " + sessionID
}

// ReadTranscript reads the raw JSONL transcript bytes for a session.
func (i *IFlowCLIAgent) ReadTranscript(sessionRef string) ([]byte, error) {
	data, err := os.ReadFile(sessionRef)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}
	return data, nil
}

// ChunkTranscript splits a JSONL transcript at line boundaries.
func (i *IFlowCLIAgent) ChunkTranscript(_ context.Context, content []byte, maxSize int) ([][]byte, error) {
	chunks, err := agent.ChunkJSONL(content, maxSize)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk JSONL transcript: %w", err)
	}
	return chunks, nil
}

// ReassembleTranscript concatenates JSONL chunks with newlines.
func (i *IFlowCLIAgent) ReassembleTranscript(chunks [][]byte) ([]byte, error) {
	return agent.ReassembleJSONL(chunks), nil
}

// GetTranscriptPosition returns the current line count of an iFlow transcript.
func (i *IFlowCLIAgent) GetTranscriptPosition(path string) (int, error) {
	if path == "" {
		return 0, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to open transcript file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	lineCount := 0

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					lineCount++
				}
				break
			}
			return 0, fmt.Errorf("failed to read transcript: %w", err)
		}
		lineCount++
	}

	return lineCount, nil
}

// ExtractModifiedFilesFromOffset extracts files modified since a given line number.
func (i *IFlowCLIAgent) ExtractModifiedFilesFromOffset(path string, startOffset int) (files []string, currentPosition int, err error) {
	if path == "" {
		return nil, 0, nil
	}

	file, openErr := os.Open(path)
	if openErr != nil {
		return nil, 0, fmt.Errorf("failed to open transcript file: %w", openErr)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var lines []TranscriptLine
	lineNum := 0

	for {
		lineData, readErr := reader.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, 0, fmt.Errorf("failed to read transcript: %w", readErr)
		}

		if len(lineData) > 0 {
			lineNum++
			if lineNum > startOffset {
				var line TranscriptLine
				if parseErr := json.Unmarshal(lineData, &line); parseErr == nil {
					lines = append(lines, line)
				}
			}
		}

		if readErr == io.EOF {
			break
		}
	}

	return ExtractModifiedFiles(lines), lineNum, nil
}

// SanitizePathForIFlow converts a path to iFlow's project directory format.
// iFlow replaces any non-alphanumeric character with a dash.
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

func SanitizePathForIFlow(path string) string {
	return nonAlphanumericRegex.ReplaceAllString(path, "-")
}
