package iflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/jsonutil"
	"github.com/entireio/cli/cmd/entire/cli/paths"
)

// Ensure IFlowCLIAgent implements HookSupport
var _ agent.HookSupport = (*IFlowCLIAgent)(nil)

// iFlow hook names - these become subcommands under `entire hooks iflow`
const (
	HookNamePreToolUse       = "pre-tool-use"
	HookNamePostToolUse      = "post-tool-use"
	HookNameSetUpEnvironment = "set-up-environment"
	HookNameStop             = "stop"
	HookNameSubagentStop     = "subagent-stop"
	HookNameSessionStart     = "session-start"
	HookNameSessionEnd       = "session-end"
	HookNameUserPromptSubmit = "user-prompt-submit"
	HookNameNotification     = "notification"
)

// HookNames returns the hook verbs iFlow CLI supports.
// These become subcommands: entire hooks iflow <verb>
func (i *IFlowCLIAgent) HookNames() []string {
	return []string{
		HookNamePreToolUse,
		HookNamePostToolUse,
		HookNameSetUpEnvironment,
		HookNameStop,
		HookNameSubagentStop,
		HookNameSessionStart,
		HookNameSessionEnd,
		HookNameUserPromptSubmit,
		HookNameNotification,
	}
}

// InstallHooks installs iFlow CLI hooks in .iflow/settings.json.
// If force is true, removes existing Entire hooks before installing.
// Returns the number of hooks installed.
func (i *IFlowCLIAgent) InstallHooks(ctx context.Context, localDev bool, force bool) (int, error) {
	// Use repo root instead of CWD to find .iflow directory
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		// Fallback to CWD if not in a git repo
		repoRoot, err = os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	settingsPath := filepath.Join(repoRoot, ".iflow", IFlowSettingsFileName)

	// Read existing settings if they exist
	var rawSettings map[string]json.RawMessage
	var rawHooks map[string]json.RawMessage
	var rawPermissions map[string]json.RawMessage

	existingData, readErr := os.ReadFile(settingsPath)
	if readErr == nil {
		if err := json.Unmarshal(existingData, &rawSettings); err != nil {
			return 0, fmt.Errorf("failed to parse existing settings.json: %w", err)
		}
		if hooksRaw, ok := rawSettings["hooks"]; ok {
			if err := json.Unmarshal(hooksRaw, &rawHooks); err != nil {
				return 0, fmt.Errorf("failed to parse hooks in settings.json: %w", err)
			}
		}
		if permRaw, ok := rawSettings["permissions"]; ok {
			if err := json.Unmarshal(permRaw, &rawPermissions); err != nil {
				return 0, fmt.Errorf("failed to parse permissions in settings.json: %w", err)
			}
		}
	} else {
		rawSettings = make(map[string]json.RawMessage)
	}

	if rawHooks == nil {
		rawHooks = make(map[string]json.RawMessage)
	}
	if rawPermissions == nil {
		rawPermissions = make(map[string]json.RawMessage)
	}

	// Parse hook types we need to modify
	var preToolUse, postToolUse, sessionStart, userPromptSubmit, notification []IFlowHookMatcher
	var setUpEnvironment, stop, subagentStop, sessionEnd []IFlowHookEntry

	parseHookMatcherType(rawHooks, "PreToolUse", &preToolUse)
	parseHookMatcherType(rawHooks, "PostToolUse", &postToolUse)
	parseHookMatcherType(rawHooks, "SessionStart", &sessionStart)
	parseHookMatcherType(rawHooks, "UserPromptSubmit", &userPromptSubmit)
	parseHookMatcherType(rawHooks, "Notification", &notification)
	parseHookEntryType(rawHooks, "SetUpEnvironment", &setUpEnvironment)
	parseHookEntryType(rawHooks, "Stop", &stop)
	parseHookEntryType(rawHooks, "SubagentStop", &subagentStop)
	parseHookEntryType(rawHooks, "SessionEnd", &sessionEnd)

	// If force is true, remove all existing Entire hooks first
	if force {
		preToolUse = removeEntireHooks(preToolUse)
		postToolUse = removeEntireHooks(postToolUse)
		sessionStart = removeEntireHooks(sessionStart)
		userPromptSubmit = removeEntireHooks(userPromptSubmit)
		notification = removeEntireHooks(notification)
		setUpEnvironment = removeEntireHookEntries(setUpEnvironment)
		stop = removeEntireHookEntries(stop)
		subagentStop = removeEntireHookEntries(subagentStop)
		sessionEnd = removeEntireHookEntries(sessionEnd)
	}

	// Define hook commands
	var preToolUseCmd, postToolUseCmd, setUpEnvCmd, stopCmd, subagentStopCmd string
	var sessionStartCmd, sessionEndCmd, userPromptSubmitCmd, notificationCmd string

	if localDev {
		baseCmd := "go run ${IFLOW_PROJECT_DIR}/cmd/entire/main.go hooks iflow"
		preToolUseCmd = baseCmd + " pre-tool-use"
		postToolUseCmd = baseCmd + " post-tool-use"
		setUpEnvCmd = baseCmd + " set-up-environment"
		stopCmd = baseCmd + " stop"
		subagentStopCmd = baseCmd + " subagent-stop"
		sessionStartCmd = baseCmd + " session-start"
		sessionEndCmd = baseCmd + " session-end"
		userPromptSubmitCmd = baseCmd + " user-prompt-submit"
		notificationCmd = baseCmd + " notification"
	} else {
		preToolUseCmd = "entire hooks iflow pre-tool-use"
		postToolUseCmd = "entire hooks iflow post-tool-use"
		setUpEnvCmd = "entire hooks iflow set-up-environment"
		stopCmd = "entire hooks iflow stop"
		subagentStopCmd = "entire hooks iflow subagent-stop"
		sessionStartCmd = "entire hooks iflow session-start"
		sessionEndCmd = "entire hooks iflow session-end"
		userPromptSubmitCmd = "entire hooks iflow user-prompt-submit"
		notificationCmd = "entire hooks iflow notification"
	}

	count := 0

	// Add PreToolUse hook with matcher "*" (all tools)
	if !hookMatcherExists(preToolUse, "*", preToolUseCmd) {
		preToolUse = addHookMatcher(preToolUse, "*", preToolUseCmd)
		count++
	}

	// Add PostToolUse hook with matcher "*" (all tools)
	if !hookMatcherExists(postToolUse, "*", postToolUseCmd) {
		postToolUse = addHookMatcher(postToolUse, "*", postToolUseCmd)
		count++
	}

	// Add SessionStart hook with matcher "startup"
	if !hookMatcherExists(sessionStart, "startup", sessionStartCmd) {
		sessionStart = addHookMatcher(sessionStart, "startup", sessionStartCmd)
		count++
	}

	// Add UserPromptSubmit hook
	if !hookMatcherHasCommand(userPromptSubmit, userPromptSubmitCmd) {
		userPromptSubmit = append(userPromptSubmit, IFlowHookMatcher{
			Hooks: []IFlowHookEntry{{Type: "command", Command: userPromptSubmitCmd}},
		})
		count++
	}

	// Add Notification hook
	if !hookMatcherHasCommand(notification, notificationCmd) {
		notification = append(notification, IFlowHookMatcher{
			Hooks: []IFlowHookEntry{{Type: "command", Command: notificationCmd}},
		})
		count++
	}

	// Add SetUpEnvironment hook
	if !hookEntryExists(setUpEnvCmd, setUpEnvironment) {
		setUpEnvironment = append(setUpEnvironment, IFlowHookEntry{
			Type:    "command",
			Command: setUpEnvCmd,
		})
		count++
	}

	// Add Stop hook
	if !hookEntryExists(stopCmd, stop) {
		stop = append(stop, IFlowHookEntry{
			Type:    "command",
			Command: stopCmd,
		})
		count++
	}

	// Add SubagentStop hook
	if !hookEntryExists(subagentStopCmd, subagentStop) {
		subagentStop = append(subagentStop, IFlowHookEntry{
			Type:    "command",
			Command: subagentStopCmd,
		})
		count++
	}

	// Add SessionEnd hook
	if !hookEntryExists(sessionEndCmd, sessionEnd) {
		sessionEnd = append(sessionEnd, IFlowHookEntry{
			Type:    "command",
			Command: sessionEndCmd,
		})
		count++
	}

	// Add permissions.deny rule if not present
	permissionsChanged := false
	var denyRules []string
	if denyRaw, ok := rawPermissions["deny"]; ok {
		if err := json.Unmarshal(denyRaw, &denyRules); err != nil {
			return 0, fmt.Errorf("failed to parse permissions.deny in settings.json: %w", err)
		}
	}
	if !slices.Contains(denyRules, metadataDenyRule) {
		denyRules = append(denyRules, metadataDenyRule)
		denyJSON, err := json.Marshal(denyRules)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal permissions.deny: %w", err)
		}
		rawPermissions["deny"] = denyJSON
		permissionsChanged = true
	}

	if count == 0 && !permissionsChanged {
		return 0, nil // All hooks and permissions already installed
	}

	// Marshal modified hook types back to rawHooks
	marshalHookMatcherType(rawHooks, "PreToolUse", preToolUse)
	marshalHookMatcherType(rawHooks, "PostToolUse", postToolUse)
	marshalHookMatcherType(rawHooks, "SessionStart", sessionStart)
	marshalHookMatcherType(rawHooks, "UserPromptSubmit", userPromptSubmit)
	marshalHookMatcherType(rawHooks, "Notification", notification)
	marshalHookEntryType(rawHooks, "SetUpEnvironment", setUpEnvironment)
	marshalHookEntryType(rawHooks, "Stop", stop)
	marshalHookEntryType(rawHooks, "SubagentStop", subagentStop)
	marshalHookEntryType(rawHooks, "SessionEnd", sessionEnd)

	// Marshal hooks and update raw settings
	hooksJSON, err := json.Marshal(rawHooks)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal hooks: %w", err)
	}
	rawSettings["hooks"] = hooksJSON

	// Marshal permissions and update raw settings
	permJSON, err := json.Marshal(rawPermissions)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal permissions: %w", err)
	}
	rawSettings["permissions"] = permJSON

	// Write back to file
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o750); err != nil {
		return 0, fmt.Errorf("failed to create .iflow directory: %w", err)
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

// UninstallHooks removes Entire hooks from iFlow CLI settings.
func (i *IFlowCLIAgent) UninstallHooks(ctx context.Context) error {
	// Use repo root to find .iflow directory when run from a subdirectory
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		repoRoot = "."
	}
	settingsPath := filepath.Join(repoRoot, ".iflow", IFlowSettingsFileName)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}

	var rawSettings map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		return fmt.Errorf("failed to parse settings.json: %w", err)
	}

	var rawHooks map[string]json.RawMessage
	if hooksRaw, ok := rawSettings["hooks"]; ok {
		if err := json.Unmarshal(hooksRaw, &rawHooks); err != nil {
			return fmt.Errorf("failed to parse hooks: %w", err)
		}
	}
	if rawHooks == nil {
		rawHooks = make(map[string]json.RawMessage)
	}

	// Parse and clean all hook types
	var preToolUse, postToolUse, sessionStart, userPromptSubmit, notification []IFlowHookMatcher
	var setUpEnvironment, stop, subagentStop, sessionEnd []IFlowHookEntry

	parseHookMatcherType(rawHooks, "PreToolUse", &preToolUse)
	parseHookMatcherType(rawHooks, "PostToolUse", &postToolUse)
	parseHookMatcherType(rawHooks, "SessionStart", &sessionStart)
	parseHookMatcherType(rawHooks, "UserPromptSubmit", &userPromptSubmit)
	parseHookMatcherType(rawHooks, "Notification", &notification)
	parseHookEntryType(rawHooks, "SetUpEnvironment", &setUpEnvironment)
	parseHookEntryType(rawHooks, "Stop", &stop)
	parseHookEntryType(rawHooks, "SubagentStop", &subagentStop)
	parseHookEntryType(rawHooks, "SessionEnd", &sessionEnd)

	// Remove Entire hooks from all hook types
	preToolUse = removeEntireHooks(preToolUse)
	postToolUse = removeEntireHooks(postToolUse)
	sessionStart = removeEntireHooks(sessionStart)
	userPromptSubmit = removeEntireHooks(userPromptSubmit)
	notification = removeEntireHooks(notification)
	setUpEnvironment = removeEntireHookEntries(setUpEnvironment)
	stop = removeEntireHookEntries(stop)
	subagentStop = removeEntireHookEntries(subagentStop)
	sessionEnd = removeEntireHookEntries(sessionEnd)

	// Marshal modified hook types back to rawHooks
	marshalHookMatcherType(rawHooks, "PreToolUse", preToolUse)
	marshalHookMatcherType(rawHooks, "PostToolUse", postToolUse)
	marshalHookMatcherType(rawHooks, "SessionStart", sessionStart)
	marshalHookMatcherType(rawHooks, "UserPromptSubmit", userPromptSubmit)
	marshalHookMatcherType(rawHooks, "Notification", notification)
	marshalHookEntryType(rawHooks, "SetUpEnvironment", setUpEnvironment)
	marshalHookEntryType(rawHooks, "Stop", stop)
	marshalHookEntryType(rawHooks, "SubagentStop", subagentStop)
	marshalHookEntryType(rawHooks, "SessionEnd", sessionEnd)

	// Also remove the metadata deny rule from permissions
	var rawPermissions map[string]json.RawMessage
	if permRaw, ok := rawSettings["permissions"]; ok {
		if err := json.Unmarshal(permRaw, &rawPermissions); err != nil {
			rawPermissions = nil
		}
	}

	if rawPermissions != nil {
		if denyRaw, ok := rawPermissions["deny"]; ok {
			var denyRules []string
			if err := json.Unmarshal(denyRaw, &denyRules); err == nil {
				filteredRules := make([]string, 0, len(denyRules))
				for _, rule := range denyRules {
					if rule != metadataDenyRule {
						filteredRules = append(filteredRules, rule)
					}
				}
				if len(filteredRules) > 0 {
					denyJSON, err := json.Marshal(filteredRules)
					if err == nil {
						rawPermissions["deny"] = denyJSON
					}
				} else {
					delete(rawPermissions, "deny")
				}
			}
		}

		if len(rawPermissions) > 0 {
			permJSON, err := json.Marshal(rawPermissions)
			if err == nil {
				rawSettings["permissions"] = permJSON
			}
		} else {
			delete(rawSettings, "permissions")
		}
	}

	// Marshal hooks back
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

// AreHooksInstalled checks if Entire hooks are currently installed.
func (i *IFlowCLIAgent) AreHooksInstalled(ctx context.Context) bool {
	repoRoot, err := paths.WorktreeRoot(ctx)
	if err != nil {
		repoRoot = "."
	}
	settingsPath := filepath.Join(repoRoot, ".iflow", IFlowSettingsFileName)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}

	var settings IFlowSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}

	// Check for at least one of our hooks
	return hookEntryExists("entire hooks iflow stop", settings.Hooks.Stop) ||
		hookEntryExists("go run ${IFLOW_PROJECT_DIR}/cmd/entire/main.go hooks iflow stop", settings.Hooks.Stop)
}

// Helper functions for hook management

func parseHookMatcherType(rawHooks map[string]json.RawMessage, hookType string, target *[]IFlowHookMatcher) {
	if data, ok := rawHooks[hookType]; ok {
		json.Unmarshal(data, target)
	}
}

func parseHookEntryType(rawHooks map[string]json.RawMessage, hookType string, target *[]IFlowHookEntry) {
	if data, ok := rawHooks[hookType]; ok {
		json.Unmarshal(data, target)
	}
}

func marshalHookMatcherType(rawHooks map[string]json.RawMessage, hookType string, matchers []IFlowHookMatcher) {
	if len(matchers) == 0 {
		delete(rawHooks, hookType)
		return
	}
	data, err := json.Marshal(matchers)
	if err != nil {
		return
	}
	rawHooks[hookType] = data
}

func marshalHookEntryType(rawHooks map[string]json.RawMessage, hookType string, entries []IFlowHookEntry) {
	if len(entries) == 0 {
		delete(rawHooks, hookType)
		return
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return
	}
	rawHooks[hookType] = data
}

func isEntireHook(command string) bool {
	for _, prefix := range entireHookPrefixes {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

func removeEntireHooks(matchers []IFlowHookMatcher) []IFlowHookMatcher {
	result := make([]IFlowHookMatcher, 0, len(matchers))
	for _, matcher := range matchers {
		filteredHooks := make([]IFlowHookEntry, 0, len(matcher.Hooks))
		for _, hook := range matcher.Hooks {
			if !isEntireHook(hook.Command) {
				filteredHooks = append(filteredHooks, hook)
			}
		}
		if len(filteredHooks) > 0 {
			matcher.Hooks = filteredHooks
			result = append(result, matcher)
		}
	}
	return result
}

func removeEntireHookEntries(entries []IFlowHookEntry) []IFlowHookEntry {
	result := make([]IFlowHookEntry, 0, len(entries))
	for _, entry := range entries {
		if !isEntireHook(entry.Command) {
			result = append(result, entry)
		}
	}
	return result
}

func hookMatcherExists(matchers []IFlowHookMatcher, matcherPattern, command string) bool {
	for _, matcher := range matchers {
		if matcher.Matcher == matcherPattern {
			for _, hook := range matcher.Hooks {
				if hook.Command == command {
					return true
				}
			}
		}
	}
	return false
}

func hookEntryExists(command string, entries []IFlowHookEntry) bool {
	for _, entry := range entries {
		if entry.Command == command {
			return true
		}
	}
	return false
}

func hookMatcherHasCommand(matchers []IFlowHookMatcher, command string) bool {
	for _, matcher := range matchers {
		for _, hook := range matcher.Hooks {
			if hook.Command == command {
				return true
			}
		}
	}
	return false
}

func addHookMatcher(matchers []IFlowHookMatcher, matcherPattern, command string) []IFlowHookMatcher {
	entry := IFlowHookEntry{
		Type:    "command",
		Command: command,
	}

	for i, matcher := range matchers {
		if matcher.Matcher == matcherPattern {
			matchers[i].Hooks = append(matchers[i].Hooks, entry)
			return matchers
		}
	}

	return append(matchers, IFlowHookMatcher{
		Matcher: matcherPattern,
		Hooks:   []IFlowHookEntry{entry},
	})
}
