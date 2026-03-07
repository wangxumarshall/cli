package iflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallHooks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Create .iflow directory
	if err := os.MkdirAll(filepath.Join(dir, ".iflow"), 0o755); err != nil {
		t.Fatal(err)
	}

	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)
	ctx := context.Background()

	// Install hooks
	count, err := ag.InstallHooks(ctx, false, false)
	if err != nil {
		t.Fatalf("InstallHooks failed: %v", err)
	}

	if count == 0 {
		t.Error("Expected hooks to be installed, got 0")
	}

	// Verify settings.json was created
	settingsPath := filepath.Join(dir, ".iflow", IFlowSettingsFileName)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}

	var settings IFlowSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to unmarshal settings.json: %v", err)
	}

	// Check that hooks were installed
	if len(settings.Hooks.Stop) == 0 {
		t.Error("Expected Stop hooks to be installed")
	}
	if len(settings.Hooks.SessionStart) == 0 {
		t.Error("Expected SessionStart hooks to be installed")
	}
	if len(settings.Hooks.PreToolUse) == 0 {
		t.Error("Expected PreToolUse hooks to be installed")
	}

	// Check permissions
	if len(settings.Permissions.Deny) == 0 {
		t.Error("Expected permissions.deny to be set")
	}
}

func TestInstallHooks_Idempotent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.MkdirAll(filepath.Join(dir, ".iflow"), 0o755); err != nil {
		t.Fatal(err)
	}

	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)
	ctx := context.Background()

	// Install hooks first time
	count1, err := ag.InstallHooks(ctx, false, false)
	if err != nil {
		t.Fatalf("First InstallHooks failed: %v", err)
	}
	if count1 == 0 {
		t.Error("Expected hooks to be installed on first run")
	}

	// Install hooks second time (should be idempotent)
	count2, err := ag.InstallHooks(ctx, false, false)
	if err != nil {
		t.Fatalf("Second InstallHooks failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("Expected 0 hooks on second install, got %d", count2)
	}
}

func TestInstallHooks_Force(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.MkdirAll(filepath.Join(dir, ".iflow"), 0o755); err != nil {
		t.Fatal(err)
	}

	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)
	ctx := context.Background()

	// Install hooks first
	_, err := ag.InstallHooks(ctx, false, false)
	if err != nil {
		t.Fatalf("First InstallHooks failed: %v", err)
	}

	// Force reinstall
	count, err := ag.InstallHooks(ctx, false, true)
	if err != nil {
		t.Fatalf("Force InstallHooks failed: %v", err)
	}
	if count == 0 {
		t.Error("Expected hooks to be reinstalled with force=true")
	}
}

func TestUninstallHooks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.MkdirAll(filepath.Join(dir, ".iflow"), 0o755); err != nil {
		t.Fatal(err)
	}

	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)
	ctx := context.Background()

	// Install hooks
	_, err := ag.InstallHooks(ctx, false, false)
	if err != nil {
		t.Fatalf("InstallHooks failed: %v", err)
	}

	// Verify hooks are installed
	if !ag.AreHooksInstalled(ctx) {
		t.Error("Expected hooks to be installed")
	}

	// Uninstall hooks
	if err := ag.UninstallHooks(ctx); err != nil {
		t.Fatalf("UninstallHooks failed: %v", err)
	}

	// Verify hooks are not installed
	if ag.AreHooksInstalled(ctx) {
		t.Error("Expected hooks to be uninstalled")
	}
}

func TestAreHooksInstalled(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)
	ctx := context.Background()

	// Initially no hooks
	if ag.AreHooksInstalled(ctx) {
		t.Error("Expected no hooks initially")
	}

	// Create .iflow directory and install hooks
	if err := os.MkdirAll(filepath.Join(dir, ".iflow"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := ag.InstallHooks(ctx, false, false)
	if err != nil {
		t.Fatalf("InstallHooks failed: %v", err)
	}

	// Now hooks should be installed
	if !ag.AreHooksInstalled(ctx) {
		t.Error("Expected hooks to be detected")
	}
}

func TestHookNames(t *testing.T) {
	ag := NewIFlowCLIAgent().(*IFlowCLIAgent)
	names := ag.HookNames()

	expected := []string{
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

	if len(names) != len(expected) {
		t.Errorf("Expected %d hook names, got %d", len(expected), len(names))
	}

	for i, name := range expected {
		if i >= len(names) || names[i] != name {
			t.Errorf("Expected hook name %q at position %d, got %q", name, i, names[i])
		}
	}
}

func TestIsEntireHook(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"entire hooks iflow stop", true},
		{"go run ${IFLOW_PROJECT_DIR}/cmd/entire/main.go hooks iflow stop", true},
		{"some other command", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := isEntireHook(tt.command)
			if result != tt.expected {
				t.Errorf("isEntireHook(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestAddHookMatcher(t *testing.T) {
	tests := []struct {
		name           string
		initial        []IFlowHookMatcher
		matcherPattern string
		command        string
		expectedLen    int
	}{
		{
			name:           "add to empty",
			initial:        []IFlowHookMatcher{},
			matcherPattern: "*",
			command:        "test command",
			expectedLen:    1,
		},
		{
			name: "add to existing matcher",
			initial: []IFlowHookMatcher{
				{Matcher: "*", Hooks: []IFlowHookEntry{{Type: "command", Command: "existing"}}},
			},
			matcherPattern: "*",
			command:        "new command",
			expectedLen:    1,
		},
		{
			name: "add new matcher",
			initial: []IFlowHookMatcher{
				{Matcher: "Edit", Hooks: []IFlowHookEntry{{Type: "command", Command: "edit hook"}}},
			},
			matcherPattern: "Write",
			command:        "write hook",
			expectedLen:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addHookMatcher(tt.initial, tt.matcherPattern, tt.command)
			if len(result) != tt.expectedLen {
				t.Errorf("Expected %d matchers, got %d", tt.expectedLen, len(result))
			}
		})
	}
}

func TestHookMatcherExists(t *testing.T) {
	matchers := []IFlowHookMatcher{
		{
			Matcher: "*",
			Hooks: []IFlowHookEntry{
				{Type: "command", Command: "entire hooks iflow stop"},
			},
		},
	}

	tests := []struct {
		matcherPattern string
		command        string
		expected       bool
	}{
		{"*", "entire hooks iflow stop", true},
		{"*", "other command", false},
		{"Edit", "entire hooks iflow stop", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := hookMatcherExists(matchers, tt.matcherPattern, tt.command)
			if result != tt.expected {
				t.Errorf("hookMatcherExists(%q, %q) = %v, want %v", tt.matcherPattern, tt.command, result, tt.expected)
			}
		})
	}
}

func TestRemoveEntireHooks(t *testing.T) {
	matchers := []IFlowHookMatcher{
		{
			Matcher: "*",
			Hooks: []IFlowHookEntry{
				{Type: "command", Command: "entire hooks iflow stop"},
				{Type: "command", Command: "other hook"},
			},
		},
		{
			Matcher: "Edit",
			Hooks: []IFlowHookEntry{
				{Type: "command", Command: "entire hooks iflow pre-tool-use"},
			},
		},
	}

	result := removeEntireHooks(matchers)

	// Should have 1 matcher (Edit matcher removed because all hooks were Entire hooks)
	if len(result) != 1 {
		t.Errorf("Expected 1 matcher after removal, got %d", len(result))
	}

	// The remaining matcher should have 1 hook (other hook)
	if len(result[0].Hooks) != 1 {
		t.Errorf("Expected 1 hook in remaining matcher, got %d", len(result[0].Hooks))
	}

	if result[0].Hooks[0].Command != "other hook" {
		t.Errorf("Expected 'other hook', got %q", result[0].Hooks[0].Command)
	}
}
