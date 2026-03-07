// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/jsonutil"
	"github.com/entireio/cli/cmd/entire/cli/paths"
)

// Ensure TraeAgent implements HookSupport
var _ agent.HookSupport = (*TraeAgent)(nil)

// Trae Agent hook names - these become subcommands under `entire hooks trae-agent`
const (
	HookNameSessionStart        = "session-start"
	HookNameSessionEnd          = "session-end"
	HookNameBeforeAgent         = "before-agent"
	HookNameAfterAgent          = "after-agent"
	HookNameBeforeModel         = "before-model"
	HookNameAfterModel          = "after-model"
	HookNameBeforeToolSelection = "before-tool-selection"
	HookNamePreTool             = "pre-tool"
	HookNameAfterTool           = "after-tool"
	HookNamePreCompress         = "pre-compress"
	HookNameNotification        = "notification"
)

// TraeSettingsFileName is the settings file used by Trae Agent.
const TraeSettingsFileName = "settings.json"

// GetHookNames returns the hook verbs Trae Agent supports.
// These become subcommands: entire hooks trae-agent <verb>
func (t *TraeAgent) HookNames() []string {
	return []string{
		HookNameSessionStart,
		HookNameSessionEnd,
		HookNameBeforeAgent,
		HookNameAfterAgent,
		HookNameBeforeModel,
		HookNameAfterModel,
		HookNameBeforeToolSelection,
		HookNamePreTool,
		HookNameAfterTool,
		HookNamePreCompress,
		HookNameNotification,
	}
}

// entireHookPrefixes are command prefixes that identify Entire hooks (both old and new formats)
var entireHookPrefixes = []string{
	"entire ",
	"go run ${TRAE_PROJECT_DIR}/cmd/entire/main.go ",
}

// InstallHooks installs Trae Agent hooks in .trae/settings.json.
// If force is true, removes existing Entire hooks before installing.
// Returns the number of hooks installed.
func (t *TraeAgent) InstallHooks(ctx context.Context, localDev bool, force bool) (int, error) {
	// Use repo root instead of CWD to find .trae directory
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		// Fallback to CWD if not in a git repo (e.g., during tests)
		repoRoot, err = os.Getwd() //nolint:forbidigo // Intentional fallback when WorktreeRoot() fails
		if err != nil {
			return 0, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	settingsPath := filepath.Join(repoRoot, ".trae", TraeSettingsFileName)

	// Read existing settings if they exist
	var rawSettings map[string]json.RawMessage

	// rawHooks preserves unknown hook types
	var rawHooks map[string]json.RawMessage

	existingData, readErr := os.ReadFile(settingsPath) //nolint:gosec // path is constructed safely
	if readErr == nil {
		if err := json.Unmarshal(existingData, &rawSettings); err != nil {
			return 0, fmt.Errorf("failed to parse existing settings.json: %w", err)
		}
		if hooksRaw, ok := rawSettings["hooks"]; ok {
			if err := json.Unmarshal(hooksRaw, &rawHooks); err != nil {
				return 0, fmt.Errorf("failed to parse hooks in settings.json: %w", err)
			}
		}
	} else {
		rawSettings = make(map[string]json.RawMessage)
	}

	if rawHooks == nil {
		rawHooks = make(map[string]json.RawMessage)
	}

	// Parse only the hook types we need to modify
	var sessionStart, sessionEnd, beforeAgent, afterAgent, beforeModel, afterModel []TraeHook
	var beforeToolSelection, preTool, afterTool, preCompress, notification []TraeHook

	parseHookType(rawHooks, "SessionStart", &sessionStart)
	parseHookType(rawHooks, "SessionEnd", &sessionEnd)
	parseHookType(rawHooks, "BeforeAgent", &beforeAgent)
	parseHookType(rawHooks, "AfterAgent", &afterAgent)
	parseHookType(rawHooks, "BeforeModel", &beforeModel)
	parseHookType(rawHooks, "AfterModel", &afterModel)
	parseHookType(rawHooks, "BeforeToolSelection", &beforeToolSelection)
	parseHookType(rawHooks, "PreTool", &preTool)
	parseHookType(rawHooks, "AfterTool", &afterTool)
	parseHookType(rawHooks, "PreCompress", &preCompress)
	parseHookType(rawHooks, "Notification", &notification)

	// If force is true, remove all existing Entire hooks first
	if force {
		sessionStart = removeEntireHooks(sessionStart)
		sessionEnd = removeEntireHooks(sessionEnd)
		beforeAgent = removeEntireHooks(beforeAgent)
		afterAgent = removeEntireHooks(afterAgent)
		beforeModel = removeEntireHooks(beforeModel)
		afterModel = removeEntireHooks(afterModel)
		beforeToolSelection = removeEntireHooks(beforeToolSelection)
		preTool = removeEntireHooks(preTool)
		afterTool = removeEntireHooks(afterTool)
		preCompress = removeEntireHooks(preCompress)
		notification = removeEntireHooks(notification)
	}

	// Define hook commands
	var sessionStartCmd, sessionEndCmd, beforeAgentCmd, afterAgentCmd string
	var beforeModelCmd, afterModelCmd, beforeToolSelectionCmd, preToolCmd string
	var afterToolCmd, preCompressCmd, notificationCmd string

	if localDev {
		baseCmd := "go run ${TRAE_PROJECT_DIR}/cmd/entire/main.go hooks trae-agent"
		sessionStartCmd = fmt.Sprintf("%s %s", baseCmd, HookNameSessionStart)
		sessionEndCmd = fmt.Sprintf("%s %s", baseCmd, HookNameSessionEnd)
		beforeAgentCmd = fmt.Sprintf("%s %s", baseCmd, HookNameBeforeAgent)
		afterAgentCmd = fmt.Sprintf("%s %s", baseCmd, HookNameAfterAgent)
		beforeModelCmd = fmt.Sprintf("%s %s", baseCmd, HookNameBeforeModel)
		afterModelCmd = fmt.Sprintf("%s %s", baseCmd, HookNameAfterModel)
		beforeToolSelectionCmd = fmt.Sprintf("%s %s", baseCmd, HookNameBeforeToolSelection)
		preToolCmd = fmt.Sprintf("%s %s", baseCmd, HookNamePreTool)
		afterToolCmd = fmt.Sprintf("%s %s", baseCmd, HookNameAfterTool)
		preCompressCmd = fmt.Sprintf("%s %s", baseCmd, HookNamePreCompress)
		notificationCmd = fmt.Sprintf("%s %s", baseCmd, HookNameNotification)
	} else {
		baseCmd := "entire hooks trae-agent"
		sessionStartCmd = fmt.Sprintf("%s %s", baseCmd, HookNameSessionStart)
		sessionEndCmd = fmt.Sprintf("%s %s", baseCmd, HookNameSessionEnd)
		beforeAgentCmd = fmt.Sprintf("%s %s", baseCmd, HookNameBeforeAgent)
		afterAgentCmd = fmt.Sprintf("%s %s", baseCmd, HookNameAfterAgent)
		beforeModelCmd = fmt.Sprintf("%s %s", baseCmd, HookNameBeforeModel)
		afterModelCmd = fmt.Sprintf("%s %s", baseCmd, HookNameAfterModel)
		beforeToolSelectionCmd = fmt.Sprintf("%s %s", baseCmd, HookNameBeforeToolSelection)
		preToolCmd = fmt.Sprintf("%s %s", baseCmd, HookNamePreTool)
		afterToolCmd = fmt.Sprintf("%s %s", baseCmd, HookNameAfterTool)
		preCompressCmd = fmt.Sprintf("%s %s", baseCmd, HookNamePreCompress)
		notificationCmd = fmt.Sprintf("%s %s", baseCmd, HookNameNotification)
	}

	count := 0

	// Add hooks if they don't exist
	if !hookCommandExists(sessionStart, sessionStartCmd) {
		sessionStart = append(sessionStart, TraeHook{Name: "entire-session-start", Type: "command", Command: sessionStartCmd})
		count++
	}
	if !hookCommandExists(sessionEnd, sessionEndCmd) {
		sessionEnd = append(sessionEnd, TraeHook{Name: "entire-session-end", Type: "command", Command: sessionEndCmd})
		count++
	}
	if !hookCommandExists(beforeAgent, beforeAgentCmd) {
		beforeAgent = append(beforeAgent, TraeHook{Name: "entire-before-agent", Type: "command", Command: beforeAgentCmd})
		count++
	}
	if !hookCommandExists(afterAgent, afterAgentCmd) {
		afterAgent = append(afterAgent, TraeHook{Name: "entire-after-agent", Type: "command", Command: afterAgentCmd})
		count++
	}
	if !hookCommandExists(beforeModel, beforeModelCmd) {
		beforeModel = append(beforeModel, TraeHook{Name: "entire-before-model", Type: "command", Command: beforeModelCmd})
		count++
	}
	if !hookCommandExists(afterModel, afterModelCmd) {
		afterModel = append(afterModel, TraeHook{Name: "entire-after-model", Type: "command", Command: afterModelCmd})
		count++
	}
	if !hookCommandExists(beforeToolSelection, beforeToolSelectionCmd) {
		beforeToolSelection = append(beforeToolSelection, TraeHook{Name: "entire-before-tool-selection", Type: "command", Command: beforeToolSelectionCmd})
		count++
	}
	if !hookCommandExists(preTool, preToolCmd) {
		preTool = append(preTool, TraeHook{Name: "entire-pre-tool", Type: "command", Command: preToolCmd})
		count++
	}
	if !hookCommandExists(afterTool, afterToolCmd) {
		afterTool = append(afterTool, TraeHook{Name: "entire-after-tool", Type: "command", Command: afterToolCmd})
		count++
	}
	if !hookCommandExists(preCompress, preCompressCmd) {
		preCompress = append(preCompress, TraeHook{Name: "entire-pre-compress", Type: "command", Command: preCompressCmd})
		count++
	}
	if !hookCommandExists(notification, notificationCmd) {
		notification = append(notification, TraeHook{Name: "entire-notification", Type: "command", Command: notificationCmd})
		count++
	}

	if count == 0 {
		return 0, nil // All hooks already installed
	}

	// Marshal modified hook types back to rawHooks
	marshalHookType(rawHooks, "SessionStart", sessionStart)
	marshalHookType(rawHooks, "SessionEnd", sessionEnd)
	marshalHookType(rawHooks, "BeforeAgent", beforeAgent)
	marshalHookType(rawHooks, "AfterAgent", afterAgent)
	marshalHookType(rawHooks, "BeforeModel", beforeModel)
	marshalHookType(rawHooks, "AfterModel", afterModel)
	marshalHookType(rawHooks, "BeforeToolSelection", beforeToolSelection)
	marshalHookType(rawHooks, "PreTool", preTool)
	marshalHookType(rawHooks, "AfterTool", afterTool)
	marshalHookType(rawHooks, "PreCompress", preCompress)
	marshalHookType(rawHooks, "Notification", notification)

	// Marshal hooks and update raw settings
	hooksJSON, err := json.Marshal(rawHooks)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal hooks: %w", err)
	}
	rawSettings["hooks"] = hooksJSON

	// Set hooksConfig.enabled = true (required for Trae Agent to execute hooks)
	hooksConfig := TraeHooksConfig{Enabled: true}
	hooksConfigJSON, err := json.Marshal(hooksConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal hooksConfig: %w", err)
	}
	rawSettings["hooksConfig"] = hooksConfigJSON

	// Write back to file
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o750); err != nil {
		return 0, fmt.Errorf("failed to create .trae directory: %w", err)
	}

	output, err := jsonutil.MarshalIndentWithNewline(rawSettings, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, output, 0o600); err != nil {
		return 0, fmt.Errorf("failed to write settings.json: %w", err)
	}

	return count, nil
}

// parseHookType parses a specific hook type from rawHooks into the target slice.
// Silently ignores parse errors (leaves target unchanged).
func parseHookType(rawHooks map[string]json.RawMessage, hookType string, target interface{}) {
	if data, ok := rawHooks[hookType]; ok {
		//nolint:errcheck,gosec // Intentionally ignoring parse errors
		json.Unmarshal(data, target)
	}
}

// marshalHookType marshals a hook type back to rawHooks.
// If the slice is empty, removes the key from rawHooks.
func marshalHookType(rawHooks map[string]json.RawMessage, hookType string, hooks interface{}) {
	// Check if hooks is empty
	var isEmpty bool
	switch h := hooks.(type) {
	case []TraeHook:
		isEmpty = len(h) == 0
	default:
		isEmpty = true
	}

	if isEmpty {
		delete(rawHooks, hookType)
		return
	}

	data, err := json.Marshal(hooks)
	if err != nil {
		return // Silently ignore marshal errors
	}
	rawHooks[hookType] = data
}

// UninstallHooks removes Entire hooks from Trae Agent settings.
func (t *TraeAgent) UninstallHooks(ctx context.Context) error {
	// Use repo root to find .trae directory when run from a subdirectory
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		repoRoot = "." // Fallback to CWD if not in a git repo
	}
	settingsPath := filepath.Join(repoRoot, ".trae", TraeSettingsFileName)
	data, err := os.ReadFile(settingsPath) //nolint:gosec // path is constructed safely
	if err != nil {
		return nil //nolint:nilerr // No settings file means nothing to uninstall
	}

	var rawSettings map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		return fmt.Errorf("failed to parse settings.json: %w", err)
	}

	// rawHooks preserves unknown hook types
	var rawHooks map[string]json.RawMessage
	if hooksRaw, ok := rawSettings["hooks"]; ok {
		if err := json.Unmarshal(hooksRaw, &rawHooks); err != nil {
			return fmt.Errorf("failed to parse hooks: %w", err)
		}
	}
	if rawHooks == nil {
		rawHooks = make(map[string]json.RawMessage)
	}

	// Parse all hook types
	var sessionStart, sessionEnd, beforeAgent, afterAgent, beforeModel, afterModel []TraeHook
	var beforeToolSelection, preTool, afterTool, preCompress, notification []TraeHook

	parseHookType(rawHooks, "SessionStart", &sessionStart)
	parseHookType(rawHooks, "SessionEnd", &sessionEnd)
	parseHookType(rawHooks, "BeforeAgent", &beforeAgent)
	parseHookType(rawHooks, "AfterAgent", &afterAgent)
	parseHookType(rawHooks, "BeforeModel", &beforeModel)
	parseHookType(rawHooks, "AfterModel", &afterModel)
	parseHookType(rawHooks, "BeforeToolSelection", &beforeToolSelection)
	parseHookType(rawHooks, "PreTool", &preTool)
	parseHookType(rawHooks, "AfterTool", &afterTool)
	parseHookType(rawHooks, "PreCompress", &preCompress)
	parseHookType(rawHooks, "Notification", &notification)

	// Remove Entire hooks from all hook types
	sessionStart = removeEntireHooks(sessionStart)
	sessionEnd = removeEntireHooks(sessionEnd)
	beforeAgent = removeEntireHooks(beforeAgent)
	afterAgent = removeEntireHooks(afterAgent)
	beforeModel = removeEntireHooks(beforeModel)
	afterModel = removeEntireHooks(afterModel)
	beforeToolSelection = removeEntireHooks(beforeToolSelection)
	preTool = removeEntireHooks(preTool)
	afterTool = removeEntireHooks(afterTool)
	preCompress = removeEntireHooks(preCompress)
	notification = removeEntireHooks(notification)

	// Marshal modified hook types back to rawHooks
	marshalHookType(rawHooks, "SessionStart", sessionStart)
	marshalHookType(rawHooks, "SessionEnd", sessionEnd)
	marshalHookType(rawHooks, "BeforeAgent", beforeAgent)
	marshalHookType(rawHooks, "AfterAgent", afterAgent)
	marshalHookType(rawHooks, "BeforeModel", beforeModel)
	marshalHookType(rawHooks, "AfterModel", afterModel)
	marshalHookType(rawHooks, "BeforeToolSelection", beforeToolSelection)
	marshalHookType(rawHooks, "PreTool", preTool)
	marshalHookType(rawHooks, "AfterTool", afterTool)
	marshalHookType(rawHooks, "PreCompress", preCompress)
	marshalHookType(rawHooks, "Notification", notification)

	// Marshal hooks back (preserving unknown hook types)
	if len(rawHooks) > 0 {
		hooksJSON, err := json.Marshal(rawHooks)
		if err != nil {
			return fmt.Errorf("failed to marshal hooks: %w", err)
		}
		rawSettings["hooks"] = hooksJSON
	} else {
		delete(rawSettings, "hooks")
	}

	// Write back
	output, err := jsonutil.MarshalIndentWithNewline(rawSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, output, 0o600); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}
	return nil
}

// AreHooksInstalled checks if Entire hooks are installed.
func (t *TraeAgent) AreHooksInstalled(ctx context.Context) bool {
	// Use repo root to find .trae directory when run from a subdirectory
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		repoRoot = "." // Fallback to CWD if not in a git repo
	}
	settingsPath := filepath.Join(repoRoot, ".trae", TraeSettingsFileName)
	data, err := os.ReadFile(settingsPath) //nolint:gosec // path is constructed safely
	if err != nil {
		return false
	}

	var settings TraeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}

	// Check for at least one of our hooks
	return hookCommandExists(settings.Hooks.SessionStart, "entire hooks trae-agent session-start") ||
		hookCommandExists(settings.Hooks.SessionStart, "go run ${TRAE_PROJECT_DIR}/cmd/entire/main.go hooks trae-agent session-start")
}

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
	prompt, _ := input.RawData["prompt"].(string)
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

// GetSupportedHooks returns the hook types Trae Agent supports.
func (t *TraeAgent) GetSupportedHooks() []agent.HookType {
	return []agent.HookType{
		agent.HookSessionStart,
		agent.HookSessionEnd,
		agent.HookBeforeAgent,
		agent.HookAfterAgent,
		agent.HookBeforeModel,
		agent.HookAfterModel,
		agent.HookBeforeToolSelection,
		agent.HookPreTool,
		agent.HookAfterTool,
		agent.HookPreCompress,
		agent.HookNotification,
	}
}

// Helper functions for hook management

// TraeHook represents a single hook configuration in Trae Agent

type TraeHook struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Command string `json:"command"`
}

// TraeHooks represents all hook configurations in Trae Agent

type TraeHooks struct {
	SessionStart        []TraeHook `json:"SessionStart,omitempty"`
	SessionEnd          []TraeHook `json:"SessionEnd,omitempty"`
	BeforeAgent         []TraeHook `json:"BeforeAgent,omitempty"`
	AfterAgent          []TraeHook `json:"AfterAgent,omitempty"`
	BeforeModel         []TraeHook `json:"BeforeModel,omitempty"`
	AfterModel          []TraeHook `json:"AfterModel,omitempty"`
	BeforeToolSelection []TraeHook `json:"BeforeToolSelection,omitempty"`
	PreTool             []TraeHook `json:"PreTool,omitempty"`
	AfterTool           []TraeHook `json:"AfterTool,omitempty"`
	PreCompress         []TraeHook `json:"PreCompress,omitempty"`
	Notification        []TraeHook `json:"Notification,omitempty"`
}

// TraeSettings represents the complete Trae Agent settings structure

type TraeSettings struct {
	HooksConfig TraeHooksConfig `json:"hooksConfig,omitempty"`
	Hooks       TraeHooks       `json:"hooks,omitempty"`
}

// TraeHooksConfig represents the hooks configuration settings
type TraeHooksConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

func hookCommandExists(hooks []TraeHook, command string) bool {
	for _, hook := range hooks {
		if hook.Command == command {
			return true
		}
	}
	return false
}

// isEntireHook checks if a command is an Entire hook (old or new format)
func isEntireHook(command string) bool {
	for _, prefix := range entireHookPrefixes {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

// removeEntireHooks removes all Entire hooks from a list of hooks
func removeEntireHooks(hooks []TraeHook) []TraeHook {
	result := make([]TraeHook, 0, len(hooks))
	for _, hook := range hooks {
		if !isEntireHook(hook.Command) {
			result = append(result, hook)
		}
	}
	return result
}
