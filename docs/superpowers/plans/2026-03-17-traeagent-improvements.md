# Trae Agent 改进实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 改进 traeagent 实现，对齐其他 agent 的代码结构，并补充缺失的测试覆盖。

**Architecture:** 从现有文件中提取类型定义到 types.go，提取生命周期逻辑到 lifecycle.go，添加完整的测试覆盖，并添加 AGENT.md 文档。

**Tech Stack:** Go 1.26, testify/assert, testify/require

---

## Chunk 1: 代码重构 - 提取类型定义和生命周期逻辑

### Task 1: 创建 types.go

**Files:**
- Create: `cmd/entire/cli/agent/traeagent/types.go`

- [ ] **Step 1: 创建 types.go 文件，包含从 traeagent.go 提取的类型**

```go
// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import "encoding/json"

// Hook 输入解析类型 - 从 traeagent.go 提取

type sessionStartRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
}

type sessionEndRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
}

type preToolUseRaw struct {
	SessionID      string          `json:"session_id"`
	TrajectoryPath string          `json:"trajectory_path"`
	ToolName       string          `json:"tool_name"`
	ToolUseID      string          `json:"tool_use_id"`
	ToolInput      json.RawMessage `json:"tool_input"`
}

type postToolUseRaw struct {
	SessionID      string          `json:"session_id"`
	TrajectoryPath string          `json:"trajectory_path"`
	ToolName       string          `json:"tool_name"`
	ToolUseID      string          `json:"tool_use_id"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolResponse   json.RawMessage `json:"tool_response"`
}

type preModelRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
	ModelName      string `json:"model_name"`
	Prompt         string `json:"prompt"`
}

type postModelRaw struct {
	SessionID      string          `json:"session_id"`
	TrajectoryPath string          `json:"trajectory_path"`
	ModelName      string          `json:"model_name"`
	Response       string          `json:"response"`
	TokenUsage     json.RawMessage `json:"token_usage"`
}

type preCompressRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
	Context        string `json:"context"`
}

type notificationRaw struct {
	SessionID        string `json:"session_id"`
	TrajectoryPath   string `json:"trajectory_path"`
	NotificationType string `json:"notification_type"`
	Message          string `json:"message"`
}

// Hook 配置类型 - 从 hooks.go 提取

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
```

- [ ] **Step 2: 运行测试验证编译通过**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && go build ./cmd/entire/cli/agent/traeagent/...`
Expected: 编译成功

- [ ] **Step 3: 从 traeagent.go 移除已提取的类型定义**

删除 traeagent.go 中第 292-347 行的内容（包括注释 `// Raw data structures for parsing hooks` 和所有类型定义 sessionStartRaw 到 notificationRaw）

- [ ] **Step 4: 从 hooks.go 移除已提取的类型定义**

删除 hooks.go 中第 502-538 行的内容（包括注释 `// Helper functions for hook management`、`// TraeHook represents...` 等注释和所有类型定义 TraeHook 到 TraeHooksConfig）

- [ ] **Step 5: 运行测试验证重构正确**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run test ./cmd/entire/cli/agent/traeagent/...`
Expected: 所有测试通过

- [ ] **Step 6: 提交 types.go 创建**

```bash
git add cmd/entire/cli/agent/traeagent/types.go
git add cmd/entire/cli/agent/traeagent/traeagent.go
git add cmd/entire/cli/agent/traeagent/hooks.go
git commit -m "refactor(traeagent): extract type definitions to types.go"
```

---

### Task 2: 创建 lifecycle.go

**Files:**
- Create: `cmd/entire/cli/agent/traeagent/lifecycle.go`

- [ ] **Step 1: 创建 lifecycle.go 文件**

```go
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
```

- [ ] **Step 2: 从 hooks.go 移除已提取的生命周期方法**

删除 hooks.go 中第 414-500 行（ParseHookEvent 到 GetSupportedHooks 方法）

- [ ] **Step 3: 修复 ParseHookInput - 添加对 HookBeforeAgent 和 HookAfterAgent 的处理**

在 `traeagent.go` 的 `ParseHookInput` 方法的 switch 语句中，在 `case agent.HookPreCompress:` 之前添加：

```go
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
```

- [ ] **Step 4: 在 types.go 中添加 beforeAgentRaw 和 afterAgentRaw 类型定义**

在 types.go 中添加：

```go
type beforeAgentRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
	Prompt         string `json:"prompt"`
}

type afterAgentRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
}
```

- [ ] **Step 5: 运行测试验证重构正确**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run test ./cmd/entire/cli/agent/traeagent/...`
Expected: 所有测试通过

- [ ] **Step 6: 提交 lifecycle.go 创建和 ParseHookInput 修复**

```bash
git add cmd/entire/cli/agent/traeagent/lifecycle.go
git add cmd/entire/cli/agent/traeagent/hooks.go
git add cmd/entire/cli/agent/traeagent/traeagent.go
git add cmd/entire/cli/agent/traeagent/types.go
git commit -m "refactor(traeagent): extract lifecycle logic and fix HookBeforeAgent/AfterAgent parsing"
```

---

### Task 3: 添加编译时接口断言

**Files:**
- Modify: `cmd/entire/cli/agent/traeagent/traeagent.go`
- Modify: `cmd/entire/cli/agent/traeagent/hooks.go`

- [ ] **Step 1: 从 hooks.go 移除现有的接口断言**

删除 hooks.go 中第 19-20 行的接口断言：
```go
// Ensure TraeAgent implements HookSupport
var _ agent.HookSupport = (*TraeAgent)(nil)
```

- [ ] **Step 2: 在 traeagent.go 中添加编译时接口断言**

在 import 块后添加：

```go
// 编译时接口断言
var _ agent.Agent = (*TraeAgent)(nil)
var _ agent.HookSupport = (*TraeAgent)(nil)
var _ agent.TranscriptAnalyzer = (*TraeAgent)(nil)
```

- [ ] **Step 3: 运行测试验证接口断言**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && go build ./cmd/entire/cli/agent/traeagent/...`
Expected: 编译成功（接口断言通过）

- [ ] **Step 4: 提交接口断言**

```bash
git add cmd/entire/cli/agent/traeagent/traeagent.go
git add cmd/entire/cli/agent/traeagent/hooks.go
git commit -m "feat(traeagent): add compile-time interface assertions"
```

---

## Chunk 2: 添加测试文件

### Task 4: 创建 hooks_test.go

**Files:**
- Create: `cmd/entire/cli/agent/traeagent/hooks_test.go`

- [ ] **Step 1: 创建 hooks_test.go 文件**

```go
package traeagent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent/testutil"
)

func TestInstallHooks_FreshInstall(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ag := &TraeAgent{}
	count, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// 11 hooks: SessionStart, SessionEnd, BeforeAgent, AfterAgent,
	// BeforeModel, AfterModel, BeforeToolSelection, PreTool, AfterTool, PreCompress, Notification
	if count != 11 {
		t.Errorf("InstallHooks() count = %d, want 11", count)
	}

	// Verify settings.json was created with hooks
	settings := readTraeSettings(t, tempDir)

	// Verify HooksConfig.Enabled is true
	if !settings.HooksConfig.Enabled {
		t.Error("hooksConfig.enabled should be true")
	}

	// Verify all hooks are present
	if len(settings.Hooks.SessionStart) != 1 {
		t.Errorf("SessionStart hooks = %d, want 1", len(settings.Hooks.SessionStart))
	}
	if len(settings.Hooks.SessionEnd) != 1 {
		t.Errorf("SessionEnd hooks = %d, want 1", len(settings.Hooks.SessionEnd))
	}
	if len(settings.Hooks.BeforeAgent) != 1 {
		t.Errorf("BeforeAgent hooks = %d, want 1", len(settings.Hooks.BeforeAgent))
	}
	if len(settings.Hooks.AfterAgent) != 1 {
		t.Errorf("AfterAgent hooks = %d, want 1", len(settings.Hooks.AfterAgent))
	}
}

func TestInstallHooks_LocalDev(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ag := &TraeAgent{}
	_, err := ag.InstallHooks(context.Background(), true, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	settings := readTraeSettings(t, tempDir)

	// Verify local dev commands use go run
	expectedCmd := "go run ${TRAE_PROJECT_DIR}/cmd/entire/main.go hooks trae-agent session-start"
	if len(settings.Hooks.SessionStart) > 0 && settings.Hooks.SessionStart[0].Command != expectedCmd {
		t.Errorf("SessionStart command = %q, want %q", settings.Hooks.SessionStart[0].Command, expectedCmd)
	}
}

func TestInstallHooks_Idempotent(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ag := &TraeAgent{}

	// First install
	count1, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("first InstallHooks() error = %v", err)
	}
	if count1 != 11 {
		t.Errorf("first InstallHooks() count = %d, want 11", count1)
	}

	// Second install should add 0 hooks
	count2, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("second InstallHooks() error = %v", err)
	}
	if count2 != 0 {
		t.Errorf("second InstallHooks() count = %d, want 0 (idempotent)", count2)
	}

	// Verify still only 1 hook per type
	settings := readTraeSettings(t, tempDir)
	if len(settings.Hooks.SessionStart) != 1 {
		t.Errorf("SessionStart hooks = %d after double install, want 1", len(settings.Hooks.SessionStart))
	}
}

func TestInstallHooks_Force(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ag := &TraeAgent{}

	// First install
	_, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("first InstallHooks() error = %v", err)
	}

	// Force reinstall should replace hooks
	count, err := ag.InstallHooks(context.Background(), false, true)
	if err != nil {
		t.Fatalf("force InstallHooks() error = %v", err)
	}
	if count != 11 {
		t.Errorf("force InstallHooks() count = %d, want 11", count)
	}
}

func TestInstallHooks_PreservesUserHooks(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// Create settings.json with existing user hooks
	writeTraeSettings(t, tempDir, `{
		"hooks": {
			"SessionStart": [
				{"name": "my-hook", "type": "command", "command": "echo hello"}
			]
		}
	}`)

	ag := &TraeAgent{}
	_, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	settings := readTraeSettings(t, tempDir)

	// Verify user hooks are preserved (should have 2: user + entire)
	if len(settings.Hooks.SessionStart) != 2 {
		t.Errorf("SessionStart hooks = %d, want 2 (user + entire)", len(settings.Hooks.SessionStart))
	}

	// Verify user hook is still there
	foundUserHook := false
	for _, hook := range settings.Hooks.SessionStart {
		if hook.Name == "my-hook" {
			foundUserHook = true
		}
	}
	if !foundUserHook {
		t.Error("user hook 'my-hook' was not preserved")
	}
}

func TestInstallHooks_PreservesUnknownHookTypes(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// Create settings with hook types we don't handle
	writeTraeSettings(t, tempDir, `{
		"hooks": {
			"FutureHook": [
				{"name": "future-hook", "type": "command", "command": "echo future"}
			]
		}
	}`)

	ag := &TraeAgent{}
	_, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Read raw hooks to verify unknown hook types are preserved
	rawHooks := testutil.ReadRawHooks(t, tempDir, ".trae")

	// Verify FutureHook is preserved
	if _, ok := rawHooks["FutureHook"]; !ok {
		t.Errorf("FutureHook type was not preserved, got keys: %v", testutil.GetKeys(rawHooks))
	}

	// Verify our hooks were also installed
	if _, ok := rawHooks["SessionStart"]; !ok {
		t.Errorf("SessionStart hook should have been installed")
	}
}

func TestInstallHooks_PreservesUnknownFields(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// Create settings.json with unknown fields
	writeTraeSettings(t, tempDir, `{
		"someOtherField": "value",
		"customConfig": {"nested": true}
	}`)

	ag := &TraeAgent{}
	_, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Read raw settings to verify unknown fields are preserved
	settingsPath := filepath.Join(tempDir, ".trae", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var rawSettings map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		t.Fatalf("failed to parse settings.json: %v", err)
	}

	if _, ok := rawSettings["someOtherField"]; !ok {
		t.Error("someOtherField was not preserved")
	}
	if _, ok := rawSettings["customConfig"]; !ok {
		t.Error("customConfig was not preserved")
	}
}

func TestUninstallHooks(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ag := &TraeAgent{}

	// First install
	_, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Verify hooks are installed
	if !ag.AreHooksInstalled(context.Background()) {
		t.Error("hooks should be installed before uninstall")
	}

	// Uninstall
	err = ag.UninstallHooks(context.Background())
	if err != nil {
		t.Fatalf("UninstallHooks() error = %v", err)
	}

	// Verify hooks are removed
	if ag.AreHooksInstalled(context.Background()) {
		t.Error("hooks should not be installed after uninstall")
	}
}

func TestUninstallHooks_NoSettingsFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ag := &TraeAgent{}

	// Should not error when no settings file exists
	err := ag.UninstallHooks(context.Background())
	if err != nil {
		t.Fatalf("UninstallHooks() should not error when no settings file: %v", err)
	}
}

func TestUninstallHooks_PreservesUserHooks(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// Create settings with both user and entire hooks
	writeTraeSettings(t, tempDir, `{
		"hooks": {
			"SessionStart": [
				{"name": "my-hook", "type": "command", "command": "echo hello"},
				{"name": "entire-session-start", "type": "command", "command": "entire hooks trae-agent session-start"}
			]
		}
	}`)

	ag := &TraeAgent{}
	err := ag.UninstallHooks(context.Background())
	if err != nil {
		t.Fatalf("UninstallHooks() error = %v", err)
	}

	settings := readTraeSettings(t, tempDir)

	// Verify only user hooks remain
	if len(settings.Hooks.SessionStart) != 1 {
		t.Errorf("SessionStart hooks = %d after uninstall, want 1 (user only)", len(settings.Hooks.SessionStart))
	}

	// Verify it's the user hook
	if len(settings.Hooks.SessionStart) > 0 && settings.Hooks.SessionStart[0].Name != "my-hook" {
		t.Error("user hook was removed during uninstall")
	}
}

func TestAreHooksInstalled(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ag := &TraeAgent{}

	// Should be false when no settings file
	if ag.AreHooksInstalled(context.Background()) {
		t.Error("AreHooksInstalled() should be false when no settings file")
	}

	// Install hooks
	_, err := ag.InstallHooks(context.Background(), false, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Should be true after installation
	if !ag.AreHooksInstalled(context.Background()) {
		t.Error("AreHooksInstalled() should be true after installation")
	}
}

// Helper functions

func readTraeSettings(t *testing.T, tempDir string) TraeSettings {
	t.Helper()
	settingsPath := filepath.Join(tempDir, ".trae", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var settings TraeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings.json: %v", err)
	}
	return settings
}

func writeTraeSettings(t *testing.T, tempDir, content string) {
	t.Helper()
	traeDir := filepath.Join(tempDir, ".trae")
	if err := os.MkdirAll(traeDir, 0o755); err != nil {
		t.Fatalf("failed to create .trae dir: %v", err)
	}
	settingsPath := filepath.Join(traeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write settings.json: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试验证**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run test ./cmd/entire/cli/agent/traeagent/...`
Expected: 所有测试通过

- [ ] **Step 3: 提交 hooks_test.go**

```bash
git add cmd/entire/cli/agent/traeagent/hooks_test.go
git commit -m "test(traeagent): add comprehensive hook tests"
```

---

### Task 5: 创建 lifecycle_test.go

**Files:**
- Create: `cmd/entire/cli/agent/traeagent/lifecycle_test.go`

- [ ] **Step 1: 创建 lifecycle_test.go 文件**

```go
package traeagent

import (
	"context"
	"strings"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

func TestParseHookEvent_SessionStart(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}
	input := `{"session_id": "test-session-123", "trajectory_path": "/tmp/test.json"}`

	event, err := ag.ParseHookEvent(context.Background(), HookNameSessionStart, strings.NewReader(input))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.Type != agent.SessionStart {
		t.Errorf("expected event type %v, got %v", agent.SessionStart, event.Type)
	}
	if event.SessionID != "test-session-123" {
		t.Errorf("expected session_id 'test-session-123', got %q", event.SessionID)
	}
	if event.SessionRef != "/tmp/test.json" {
		t.Errorf("expected session_ref '/tmp/test.json', got %q", event.SessionRef)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestParseHookEvent_BeforeAgent(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}
	input := `{"session_id": "sess-456", "trajectory_path": "/tmp/t.json", "prompt": "Hello Trae"}`

	event, err := ag.ParseHookEvent(context.Background(), HookNameBeforeAgent, strings.NewReader(input))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.Type != agent.TurnStart {
		t.Errorf("expected event type %v, got %v", agent.TurnStart, event.Type)
	}
	if event.SessionID != "sess-456" {
		t.Errorf("expected session_id 'sess-456', got %q", event.SessionID)
	}
	if event.Prompt != "Hello Trae" {
		t.Errorf("expected prompt 'Hello Trae', got %q", event.Prompt)
	}
}

func TestParseHookEvent_AfterAgent(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}
	input := `{"session_id": "sess-789", "trajectory_path": "/tmp/after.json"}`

	event, err := ag.ParseHookEvent(context.Background(), HookNameAfterAgent, strings.NewReader(input))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.Type != agent.TurnEnd {
		t.Errorf("expected event type %v, got %v", agent.TurnEnd, event.Type)
	}
	if event.SessionID != "sess-789" {
		t.Errorf("expected session_id 'sess-789', got %q", event.SessionID)
	}
}

func TestParseHookEvent_SessionEnd(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}
	input := `{"session_id": "ending-session", "trajectory_path": "/tmp/end.json"}`

	event, err := ag.ParseHookEvent(context.Background(), HookNameSessionEnd, strings.NewReader(input))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.Type != agent.SessionEnd {
		t.Errorf("expected event type %v, got %v", agent.SessionEnd, event.Type)
	}
	if event.SessionID != "ending-session" {
		t.Errorf("expected session_id 'ending-session', got %q", event.SessionID)
	}
}

func TestParseHookEvent_UnknownHook_ReturnsNil(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}
	input := `{"session_id": "unknown", "trajectory_path": "/tmp/unknown.json"}`

	event, err := ag.ParseHookEvent(context.Background(), "unknown-hook-name", strings.NewReader(input))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for unknown hook, got %+v", event)
	}
}

func TestParseHookEvent_PassThroughHooks_ReturnNil(t *testing.T) {
	t.Parallel()

	passThroughHooks := []string{
		HookNameBeforeModel,
		HookNameAfterModel,
		HookNameBeforeToolSelection,
		HookNamePreTool,
		HookNameAfterTool,
		HookNamePreCompress,
		HookNameNotification,
	}

	ag := &TraeAgent{}
	input := `{"session_id": "test", "trajectory_path": "/t"}`

	for _, hookName := range passThroughHooks {
		t.Run(hookName, func(t *testing.T) {
			t.Parallel()

			event, err := ag.ParseHookEvent(context.Background(), hookName, strings.NewReader(input))

			if err != nil {
				t.Fatalf("unexpected error for %s: %v", hookName, err)
			}
			if event != nil {
				t.Errorf("expected nil event for %s, got %+v", hookName, event)
			}
		})
	}
}

func TestParseHookEvent_InvalidJSON(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}
	input := `{"session_id": "test", "trajectory_path": INVALID}`

	_, err := ag.ParseHookEvent(context.Background(), HookNameSessionStart, strings.NewReader(input))

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseHookEvent_EmptyInput(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}

	_, err := ag.ParseHookEvent(context.Background(), HookNameSessionStart, strings.NewReader(""))

	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}
```

- [ ] **Step 2: 运行测试验证**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run test ./cmd/entire/cli/agent/traeagent/...`
Expected: 所有测试通过

- [ ] **Step 3: 提交 lifecycle_test.go**

```bash
git add cmd/entire/cli/agent/traeagent/lifecycle_test.go
git commit -m "test(traeagent): add lifecycle event tests"
```

---

### Task 6: 创建 transcript_test.go

**Files:**
- Create: `cmd/entire/cli/agent/traeagent/transcript_test.go`

- [ ] **Step 1: 创建 transcript_test.go 文件**

```go
package traeagent

import (
	"context"
	"testing"
)

func TestParseTrajectory(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"session_id": "test-session",
		"start_time": "2024-01-01T00:00:00Z",
		"events": [
			{"type": "user_message", "content": {"text": "hello"}},
			{"type": "assistant_message", "content": {"text": "hi there"}}
		]
	}`)

	trajectory, err := ParseTrajectory(data)
	if err != nil {
		t.Fatalf("ParseTrajectory() error = %v", err)
	}

	if trajectory.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want 'test-session'", trajectory.SessionID)
	}
	if len(trajectory.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(trajectory.Events))
	}
}

func TestParseTrajectory_InvalidJSON(t *testing.T) {
	t.Parallel()

	data := []byte(`{invalid json}`)

	_, err := ParseTrajectory(data)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestExtractModifiedFiles(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"session_id": "test",
		"events": [
			{"type": "tool_execution", "tool_name": "write_file", "tool_input": {"file_path": "foo.go"}},
			{"type": "tool_execution", "tool_name": "edit_tool", "tool_input": {"file_path": "bar.go"}},
			{"type": "tool_execution", "tool_name": "bash", "tool_input": {"command": "ls"}},
			{"type": "tool_execution", "tool_name": "write_file", "tool_input": {"file_path": "foo.go"}}
		]
	}`)

	files, err := ExtractModifiedFiles(data)
	if err != nil {
		t.Fatalf("ExtractModifiedFiles() error = %v", err)
	}

	// Should have foo.go and bar.go (deduplicated, bash not included)
	if len(files) != 2 {
		t.Errorf("ExtractModifiedFiles() got %d files, want 2", len(files))
	}

	hasFile := func(name string) bool {
		for _, f := range files {
			if f == name {
				return true
			}
		}
		return false
	}

	if !hasFile("foo.go") {
		t.Error("missing foo.go")
	}
	if !hasFile("bar.go") {
		t.Error("missing bar.go")
	}
}

func TestExtractModifiedFiles_EmptyTrajectory(t *testing.T) {
	t.Parallel()

	data := []byte(`{"session_id": "empty", "events": []}`)

	files, err := ExtractModifiedFiles(data)
	if err != nil {
		t.Fatalf("ExtractModifiedFiles() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("ExtractModifiedFiles() got %d files, want 0", len(files))
	}
}

func TestChunkTranscript_SmallContent(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}
	content := []byte(`{"session_id": "test", "events": []}`)

	chunks, err := ag.ChunkTranscript(context.Background(), content, 1000)
	if err != nil {
		t.Fatalf("ChunkTranscript() error = %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("ChunkTranscript() got %d chunks, want 1", len(chunks))
	}
}

func TestReassembleTranscript(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}

	// Create two chunks
	chunk1 := []byte(`{"session_id": "test", "start_time": "2024-01-01T00:00:00Z", "events": [{"type": "user"}]}`)
	chunk2 := []byte(`{"session_id": "test", "start_time": "2024-01-01T00:00:00Z", "events": [{"type": "assistant"}]}`)

	result, err := ag.ReassembleTranscript([][]byte{chunk1, chunk2})
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	// Parse result to verify
	trajectory, err := ParseTrajectory(result)
	if err != nil {
		t.Fatalf("ParseTrajectory() error = %v", err)
	}

	if len(trajectory.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(trajectory.Events))
	}
}

func TestReassembleTranscript_EmptyChunks(t *testing.T) {
	t.Parallel()

	ag := &TraeAgent{}

	result, err := ag.ReassembleTranscript(nil)
	if err != nil {
		t.Fatalf("ReassembleTranscript() error = %v", err)
	}

	if string(result) != "{}" {
		t.Errorf("ReassembleTranscript() = %q, want '{}'", string(result))
	}
}

func TestExtractAllUserPrompts(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"session_id": "test",
		"events": [
			{"type": "user_message", "content": {"text": "Hello"}},
			{"type": "assistant_message", "content": {"text": "Hi"}},
			{"type": "user_message", "content": {"text": "How are you?"}}
		]
	}`)

	prompts, err := ExtractAllUserPrompts(data)
	if err != nil {
		t.Fatalf("ExtractAllUserPrompts() error = %v", err)
	}

	if len(prompts) != 2 {
		t.Errorf("ExtractAllUserPrompts() got %d prompts, want 2", len(prompts))
	}

	if prompts[0] != "Hello" {
		t.Errorf("prompts[0] = %q, want 'Hello'", prompts[0])
	}
	if prompts[1] != "How are you?" {
		t.Errorf("prompts[1] = %q, want 'How are you?'", prompts[1])
	}
}

func TestExtractLastAssistantMessage(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"session_id": "test",
		"events": [
			{"type": "assistant_message", "content": {"text": "First response"}},
			{"type": "user_message", "content": {"text": "Question"}},
			{"type": "assistant_message", "content": {"text": "Last response"}}
		]
	}`)

	msg, err := ExtractLastAssistantMessage(data)
	if err != nil {
		t.Fatalf("ExtractLastAssistantMessage() error = %v", err)
	}

	if msg != "Last response" {
		t.Errorf("ExtractLastAssistantMessage() = %q, want 'Last response'", msg)
	}
}

func TestExtractLastAssistantMessage_NoAssistantMessages(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"session_id": "test",
		"events": [
			{"type": "user_message", "content": {"text": "Hello"}}
		]
	}`)

	msg, err := ExtractLastAssistantMessage(data)
	if err != nil {
		t.Fatalf("ExtractLastAssistantMessage() error = %v", err)
	}

	if msg != "" {
		t.Errorf("ExtractLastAssistantMessage() = %q, want empty string", msg)
	}
}
```

- [ ] **Step 2: 运行测试验证**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run test ./cmd/entire/cli/agent/traeagent/...`
Expected: 所有测试通过

- [ ] **Step 3: 提交 transcript_test.go**

```bash
git add cmd/entire/cli/agent/traeagent/transcript_test.go
git commit -m "test(traeagent): add transcript parsing tests"
```

---

## Chunk 3: 添加文档

### Task 7: 创建 AGENT.md

**Files:**
- Create: `cmd/entire/cli/agent/traeagent/AGENT.md`

- [ ] **Step 1: 创建 AGENT.md 文件**

```markdown
# Trae Agent 集成

## 概述

Trae Agent 是字节跳动的 LLM 软件工程代理。Entire CLI 通过 hooks 集成 Trae Agent，实现会话跟踪和检查点管理。

## 支持的 Hook 类型

| Hook | 事件类型 | 说明 |
|------|----------|------|
| session-start | SessionStart | 会话开始 |
| session-end | SessionEnd | 会话结束 |
| before-agent | TurnStart | Turn 开始 |
| after-agent | TurnEnd | Turn 结束 |
| before-model | - | 模型调用前（透传） |
| after-model | - | 模型调用后（透传） |
| before-tool-selection | - | 工具选择前（透传） |
| pre-tool | - | 工具执行前（透传） |
| after-tool | - | 工具执行后（透传） |
| pre-compress | - | 上下文压缩前（透传） |
| notification | - | 通知事件（透传） |

## 配置文件位置

- `.trae/settings.json` - Hook 配置

## Session 存储位置

- `~/.trae/projects/<repo-name>/` - Session transcripts

## 已实现接口

- **Agent** (核心) - 基础代理接口
- **HookSupport** (Hook 管理) - 安装/卸载/检测 hooks
- **TranscriptAnalyzer** (Transcript 解析) - 解析 JSON trajectory

## Transcript 格式

Trae Agent 使用 JSON 格式的 trajectory 文件：

```json
{
  "session_id": "session-uuid",
  "start_time": "2024-01-01T00:00:00Z",
  "end_time": "2024-01-01T01:00:00Z",
  "events": [
    {"type": "user_message", "content": {"text": "..."}},
    {"type": "assistant_message", "content": {"text": "..."}},
    {"type": "tool_execution", "tool_name": "write_file", "tool_input": {"file_path": "..."}}
  ]
}
```

## 文件修改工具

以下工具会被识别为文件修改操作：

- `str_replace_based_edit_tool`
- `edit_tool`
- `write_file`
- `update_file`
- `delete_file`

## 已知限制

- **不支持 TokenCalculator** - 待 Trae Agent 提供 token 信息
- **不支持 HookResponseWriter** - 待 Trae Agent 支持响应输出
- **不支持 SubagentAwareExtractor** - 待确认 Trae Agent 是否支持子代理
```

- [ ] **Step 2: 提交 AGENT.md**

```bash
git add cmd/entire/cli/agent/traeagent/AGENT.md
git commit -m "docs(traeagent): add AGENT.md integration documentation"
```

---

## Chunk 4: 验证和清理

### Task 8: 运行完整测试验证

- [ ] **Step 1: 运行所有测试**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run test ./cmd/entire/cli/agent/traeagent/...`
Expected: 所有测试通过

- [ ] **Step 2: 运行代码检查**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run lint`
Expected: 无错误

- [ ] **Step 3: 运行格式化**

Run: `cd /Users/wangxu/1-project/entireio-cli/cli && mise run fmt`
Expected: 格式化完成

- [ ] **Step 4: 验证文件结构**

Run: `ls -la cmd/entire/cli/agent/traeagent/`
Expected: 包含以下文件：
- traeagent.go
- types.go
- lifecycle.go
- hooks.go
- transcript.go
- traeagent_test.go
- hooks_test.go
- lifecycle_test.go
- transcript_test.go
- AGENT.md

- [ ] **Step 5: 最终提交（如有遗漏）**

```bash
git status
# 如有未提交的更改，提交它们
git add -A
git commit -m "chore(traeagent): finalize improvements"
```

---

## 验收标准

- [ ] 所有测试通过 (`mise run test`)
- [ ] 代码检查通过 (`mise run lint`)
- [ ] 文件结构与 claudecode/geminicli 一致
- [ ] 测试覆盖 > 80%
