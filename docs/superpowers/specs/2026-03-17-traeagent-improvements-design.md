# Trae Agent 改进设计

## 概述

改进 traeagent 实现，对齐其他 agent（claudecode, geminicli）的代码结构，并补充缺失的测试覆盖。

## 目标

1. **测试覆盖**：从 ~10% 提升到 ~80%+
2. **代码组织**：分离类型定义和生命周期逻辑
3. **文档完善**：添加 AGENT.md 集成说明

## 文件结构变更

### 当前结构

```
traeagent/
├── traeagent.go      (348 行 - 主实现 + 类型定义 + 生命周期)
├── hooks.go          (569 行 - Hook 管理 + 类型定义)
├── transcript.go     (346 行 - Transcript 解析)
└── traeagent_test.go (52 行 - 仅 4 个测试)
```

### 目标结构

```
traeagent/
├── traeagent.go      (~200 行 - 核心接口实现)
├── types.go          (~150 行 - 类型定义)
├── lifecycle.go      (~100 行 - 生命周期事件处理)
├── hooks.go          (~400 行 - Hook 安装/卸载)
├── transcript.go     (~350 行 - Transcript 解析)
├── traeagent_test.go (~50 行 - 基础测试)
├── hooks_test.go     (~300 行 - Hook 测试)
├── lifecycle_test.go (~150 行 - 生命周期测试)
├── transcript_test.go(~200 行 - Transcript 测试)
└── AGENT.md          (~100 行 - 集成文档)
```

## 详细设计

### 1. types.go（新增）

从 `traeagent.go` 和 `hooks.go` 中提取类型定义：

**从 traeagent.go 提取**：
```go
// Hook 输入解析类型
type sessionStartRaw struct {
    SessionID      string `json:"session_id"`
    TrajectoryPath string `json:"trajectory_path"`
}

type sessionEndRaw struct { ... }
type preToolUseRaw struct { ... }
type postToolUseRaw struct { ... }
type preModelRaw struct { ... }
type postModelRaw struct { ... }
type preCompressRaw struct { ... }
type notificationRaw struct { ... }
```

**从 hooks.go 提取**：
```go
// Hook 配置类型
type TraeHook struct {
    Name    string `json:"name"`
    Type    string `json:"type"`
    Command string `json:"command"`
}

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

type TraeSettings struct {
    HooksConfig TraeHooksConfig `json:"hooksConfig,omitempty"`
    Hooks       TraeHooks       `json:"hooks,omitempty"`
}

type TraeHooksConfig struct {
    Enabled bool `json:"enabled,omitempty"`
}
```

**保留在 transcript.go**：
```go
// Transcript 相关类型 - 保留在 transcript.go 中（与 transcript 解析逻辑紧密相关）
type TrajectoryEvent struct { ... }
type Trajectory struct { ... }
```

> 注意：`TrajectoryEvent` 和 `Trajectory` 类型保留在 `transcript.go` 中，因为它们与 transcript 解析逻辑紧密相关。`types.go` 仅包含 Hook 配置和输入解析类型。

### 2. lifecycle.go（新增）

从 `hooks.go` 中提取生命周期事件处理逻辑：

> 注意：`ParseHookInput` 保留在 `traeagent.go` 中（属于核心 Agent 接口），`ParseHookEvent` 及其解析方法移动到 `lifecycle.go`。

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
        return nil, nil
    }
}

func (t *TraeAgent) parseSessionStart(stdin io.Reader) (*agent.Event, error) { ... }
func (t *TraeAgent) parseBeforeAgent(stdin io.Reader) (*agent.Event, error) { ... }
func (t *TraeAgent) parseAfterAgent(stdin io.Reader) (*agent.Event, error) { ... }
func (t *TraeAgent) parseSessionEnd(stdin io.Reader) (*agent.Event, error) { ... }
```

### 3. traeagent.go 修改

- 移除类型定义（移至 types.go）
- 添加编译时接口断言
- 保持核心接口实现

```go
// 编译时接口断言
var _ agent.Agent = (*TraeAgent)(nil)
var _ agent.HookSupport = (*TraeAgent)(nil)
var _ agent.TranscriptAnalyzer = (*TraeAgent)(nil)
```

### 4. hooks.go 修改

- 移除类型定义（移至 types.go）
- 移除 ParseHookEvent 相关方法（移至 lifecycle.go）
- 保持 Hook 安装/卸载逻辑

### 5. 测试文件

#### hooks_test.go

参考 geminicli 的测试模式：

```go
package traeagent

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestInstallHooks_FreshInstall(t *testing.T) {
    t.Parallel()
    tempDir := t.TempDir()
    t.Chdir(tempDir)

    ag := &TraeAgent{}
    count, err := ag.InstallHooks(context.Background(), false, false)
    require.NoError(t, err)
    assert.Equal(t, 11, count) // 11 hook types

    // 验证 settings.json 创建
    settingsPath := filepath.Join(tempDir, ".trae", "settings.json")
    _, err = os.Stat(settingsPath)
    assert.NoError(t, err)
}

func TestInstallHooks_Idempotent(t *testing.T) { ... }
func TestInstallHooks_Force(t *testing.T) { ... }
func TestInstallHooks_PreservesUserHooks(t *testing.T) { ... }
func TestInstallHooks_PreservesUnknownFields(t *testing.T) { ... }
func TestUninstallHooks(t *testing.T) { ... }
func TestAreHooksInstalled(t *testing.T) { ... }
```

#### lifecycle_test.go

参考 claudecode 的测试模式：

```go
package traeagent

import (
    "context"
    "strings"
    "testing"

    "github.com/entireio/cli/cmd/entire/cli/agent"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParseHookEvent_SessionStart(t *testing.T) {
    t.Parallel()
    ag := &TraeAgent{}
    input := `{"session_id": "test-123", "trajectory_path": "/tmp/test.json"}`

    event, err := ag.ParseHookEvent(context.Background(), HookNameSessionStart, strings.NewReader(input))
    require.NoError(t, err)
    require.NotNil(t, event)

    assert.Equal(t, agent.SessionStart, event.Type)
    assert.Equal(t, "test-123", event.SessionID)
    assert.Equal(t, "/tmp/test.json", event.SessionRef)
}

func TestParseHookEvent_BeforeAgent(t *testing.T) { ... }
func TestParseHookEvent_AfterAgent(t *testing.T) { ... }
func TestParseHookEvent_SessionEnd(t *testing.T) { ... }
func TestParseHookEvent_InvalidJSON(t *testing.T) { ... }
func TestParseHookEvent_UnknownHook(t *testing.T) { ... }
```

#### transcript_test.go

```go
package traeagent

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParseTrajectory(t *testing.T) {
    t.Parallel()
    data := []byte(`{
        "session_id": "test-session",
        "start_time": "2024-01-01T00:00:00Z",
        "events": [
            {"type": "user_message", "content": {"text": "hello"}}
        ]
    }`)

    trajectory, err := ParseTrajectory(data)
    require.NoError(t, err)
    assert.Equal(t, "test-session", trajectory.SessionID)
    assert.Len(t, trajectory.Events, 1)
}

func TestExtractModifiedFiles(t *testing.T) { ... }
func TestChunkTranscript(t *testing.T) { ... }
func TestReassembleTranscript(t *testing.T) { ... }
func TestExtractAllUserPrompts(t *testing.T) { ... }
```

### 6. AGENT.md（新增）

```markdown
# Trae Agent 集成

## 概述

Trae Agent 是字节跳动的 LLM 软件工程代理。Entire CLI 通过 hooks 集成 Trae Agent。

## 支持的 Hook 类型

| Hook | 类型 | 说明 |
|------|------|------|
| session-start | SessionStart | 会话开始 |
| session-end | SessionEnd | 会话结束 |
| before-agent | TurnStart | Turn 开始 |
| after-agent | TurnEnd | Turn 结束 |
| before-model | - | 模型调用前 |
| after-model | - | 模型调用后 |
| pre-tool | - | 工具执行前 |
| after-tool | - | 工具执行后 |
| pre-compress | - | 上下文压缩前 |
| notification | - | 通知事件 |

## 配置文件位置

- `.trae/settings.json` - Hook 配置

## Session 存储位置

- `~/.trae/projects/<repo-name>/` - Session transcripts

## 已实现接口

- Agent (核心)
- HookSupport (Hook 管理)
- TranscriptAnalyzer (Transcript 解析)

## 已知限制

- 不支持 TokenCalculator (待 Trae Agent 提供 token 信息)
- 不支持 HookResponseWriter (待 Trae Agent 支持响应输出)
```

## 实施步骤

1. **创建 types.go** - 提取类型定义
2. **创建 lifecycle.go** - 提取生命周期逻辑
3. **修改 traeagent.go** - 添加接口断言，移除已提取代码
4. **修改 hooks.go** - 移除已提取代码
5. **创建 hooks_test.go** - Hook 测试
6. **创建 lifecycle_test.go** - 生命周期测试
7. **创建 transcript_test.go** - Transcript 测试
8. **创建 AGENT.md** - 集成文档
9. **运行测试验证** - `mise run test`

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 代码移动引入错误 | 中 | 测试先行，逐步验证 |
| 接口断言失败 | 低 | 已验证接口实现完整 |
| 测试覆盖不足 | 低 | 参考其他 agent 测试模式 |

## 验收标准

- [ ] 所有测试通过 (`mise run test`)
- [ ] 代码检查通过 (`mise run lint`)
- [ ] 测试覆盖率 > 80%
- [ ] 文件结构与 claudecode/geminicli 一致
