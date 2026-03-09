# Entire CLI 架构概述

## 1. 项目概览

Entire CLI 是一个与 Git 工作流集成的工具，用于捕获和管理 AI 代理（Claude Code、Gemini CLI、Cursor、OpenCode 等）的会话数据。核心功能：

- 记录 AI 代理的完整交互过程（提示词、响应、修改的文件）
- 提供会话恢复和回滚功能
- 在单独的分支上保存会话元数据，保持主分支 Git 历史整洁
- 支持审计和合规要求

**技术栈**: Go 1.26.x, Cobra CLI, go-git/v6, mise 构建工具

## 2. 构建与命令

### 开发命令
```bash
mise install              # 安装依赖
mise run build            # 构建 CLI
mise run fmt              # 代码格式化
mise run lint             # 代码检查
```

### 测试命令
```bash
mise run test             # 单元测试
mise run test:integration # 集成测试
mise run test:ci          # CI 完整测试（单元+集成+E2E canary）
mise run test:e2e:canary  # E2E canary 测试（Vogon 代理，无 API 调用）
```

### 提交前必须执行
```bash
mise run fmt && mise run lint && mise run test:ci
```

## 3. 核心架构

### Agent 系统 (`cmd/entire/cli/agent/`)

Agent 系统使用**控制反转**模式：代理是被动数据提供者，将原生 hook 载荷转换为标准化生命周期事件，框架负责所有编排。

**核心接口** (`agent.Agent`):
- `Name()` / `Type()` / `Description()` - 身份标识
- `DetectPresence()` - 检测代理是否配置
- `ProtectedDirs()` - 回滚时保护的目录
- `ReadTranscript()` / `ChunkTranscript()` / `ReassembleTranscript()` - 会话记录处理
- `ParseHookEvent()` - **核心贡献点** - 将原生 hook 转换为标准化事件

**可选接口**:
- `HookSupport` - 自动安装 hooks 到代理配置文件
- `TranscriptAnalyzer` - 从会话记录提取文件列表和提示词
- `TokenCalculator` - 计算 token 使用量
- `SubagentAwareExtractor` - 处理子代理贡献

**已支持的代理**: Claude Code, Gemini CLI, Cursor, OpenCode, Factory AI Droid, Vogon (测试用)

### Strategy 系统 (`cmd/entire/cli/strategy/`)

Manual-Commit 策略管理会话数据和检查点：
- 创建影子分支 `entire/<commit-hash[:7]>-<worktreeHash[:6]>` 存储会话元数据
- 用户提交时压缩到 `entire/checkpoints/v1` 分支
- 支持 Git worktrees，每个 worktree 独立跟踪
- 不修改活跃分支历史，安全用于 main/master

**关键文件**:
- `strategy.go` - 接口定义和上下文结构
- `manual_commit.go` - 主实现
- `manual_commit_session.go` - 会话状态管理
- `manual_commit_condensation.go` - 压缩逻辑
- `manual_commit_rewind.go` - 回滚实现

### Checkpoint 系统 (`cmd/entire/cli/checkpoint/`)

- `temporary.go` - 影子分支操作
- `committed.go` - 元数据分支操作
- 使用 12 位十六进制检查点 ID 进行双向链接

### Session 状态机 (`cmd/entire/cli/session/`)

**阶段**: `ACTIVE` → `IDLE` → `ENDED`

**事件**: `TurnStart`, `TurnEnd`, `GitCommit`, `SessionStart`, `SessionEnd`

## 4. 代码风格

### Go 代码规范
- 遵循 `golangci-lint` 检查规则
- 使用 `slog` 进行结构化日志记录
- 所有测试使用 `t.Parallel()` 并行执行（除非修改进程全局状态）

### 错误处理模式
- `root.go` 设置 `SilenceErrors: true`，Cobra 不打印错误
- `main.go` 打印错误到 stderr，除非是 `SilentError`
- 使用 `NewSilentError()` 当需要自定义用户友好错误消息时

### Settings 访问
```go
import "github.com/entireio/cli/cmd/entire/cli/settings"

s, err := settings.Load(ctx)
if s.Enabled { ... }
```
**禁止**: 直接读取 `.entire/settings.json`

### 日志 vs 用户输出
- **内部日志**: `logging.Debug/Info/Warn/Error(ctx, msg, attrs...)` → `.entire/logs/`
- **用户输出**: `fmt.Fprint*(cmd.OutOrStdout(), ...)`
- **隐私**: 不记录用户内容（提示词、文件内容、提交消息）

### Git 操作注意事项

**禁止使用 go-git v5/v6 进行 `checkout` 或 `reset --hard`**：
go-git 有 bug 会错误删除 `.gitignore` 中列出的未跟踪目录。使用 git CLI：
```go
// 正确方式
cmd := exec.CommandContext(ctx, "git", "reset", "--hard", hash.String())
```

**路径处理**：
始终使用仓库根目录而非 `os.Getwd()`：
```go
// 错误 - 从子目录运行时会出错
cwd, _ := os.Getwd()
absPath := filepath.Join(cwd, file)

// 正确
repoRoot, _ := paths.WorktreeRoot()
absPath := filepath.Join(repoRoot, file)
```

## 5. 测试

### 测试隔离
**测试必须使用隔离的临时仓库**，禁止使用真实仓库 CWD：
```go
tmpDir := t.TempDir()
testutil.InitRepo(t, tmpDir)                    // git init + user config + disable GPG
testutil.WriteFile(t, tmpDir, "f.txt", "init")
testutil.GitAdd(t, tmpDir, "f.txt")
testutil.GitCommit(t, tmpDir, "init")
t.Chdir(tmpDir)
```

### 测试文件位置
- 单元测试: 与源文件同目录 `*_test.go`
- 集成测试: `cmd/entire/cli/integration_test/` (使用 `//go:build integration` 标签)
- E2E 测试: `e2e/tests/` (使用 `//go:build e2e` 标签)

### 代码重复检查
```bash
mise run dup           # 综合检查（阈值 50）
mise run dup:staged    # 仅检查暂存文件
```

## 6. 安全

### 数据保护
- `settings.local.json` 默认不提交到 Git
- 会话元数据存储在单独分支，与主分支分离
- `redact/` 包处理敏感信息

### 权限控制
- 设置文件使用权限 `0o644`
- 遵循最小权限原则

### 遥测
- 支持匿名使用统计，可通过配置关闭
- 非阻塞方式收集，不影响核心功能

## 7. 配置

### 配置文件
位于 `.entire/` 目录：

**settings.json** (项目设置，提交到 Git):
```json
{
  "enabled": true,
  "commit_linking": "prompt",
  "strategy_options": {
    "push_sessions": true,
    "summarize": { "enabled": true }
  }
}
```

**settings.local.json** (本地设置，不提交):
```json
{
  "enabled": false,
  "log_level": "debug"
}
```

### 主要配置选项

| 选项 | 值 | 描述 |
|------|-----|------|
| `enabled` | `true`, `false` | 启用/禁用 Entire |
| `log_level` | `debug`, `info`, `warn`, `error` | 日志级别 |
| `commit_linking` | `always`, `prompt` | 提交链接模式 |
| `strategy_options.push_sessions` | `true`, `false` | Git push 时自动推送 checkpoints 分支 |
| `strategy_options.summarize.enabled` | `true`, `false` | 自动生成 AI 摘要 |
| `telemetry` | `true`, `false` | 发送匿名使用统计 |
| `external_agents` | `true`, `false` | 启用外部代理插件发现 |

## 8. 无障碍支持

环境变量 `ACCESSIBLE=1` 启用无障碍模式，使用简单文本提示替代交互式 TUI 元素。

在 `cli` 包中使用 `NewAccessibleForm()`，在 `strategy` 包中使用 `isAccessibleMode()` 辅助函数。

## 9. 添加新代理

参考 `docs/architecture/agent-guide.md` 和 `docs/architecture/agent-integration-checklist.md`。

关键步骤：
1. 创建 `cmd/entire/cli/agent/youragent/` 目录
2. 实现 `agent.Agent` 接口（19 个方法）
3. 实现 `ParseHookEvent()` - 核心贡献点
4. 根据需要实现可选接口
5. 通过 `init()` 注册代理
6. 添加编译时接口断言
7. 编写测试

## 10. 行为规则

### 开发流程
- 非平凡任务（3+ 步骤或架构决策）先进入计划模式
- 使用子代理保持主上下文窗口清洁
- 任务完成后验证正确性（运行测试、检查日志）
- 追求优雅但不过度工程化

### 任务管理
1. 先写计划到 `tasks/todo.md`
2. 开始实现前确认计划
3. 进度跟踪：完成一项标记一项
4. 记录结果和经验教训

### 核心原则
- **简洁优先**: 每个改动尽可能简单，影响最小代码
- **不偷懒**: 找到根本原因，不做临时修复
- **最小影响**: 只触及必要的代码
