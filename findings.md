# Findings & Decisions

## Network/Build Issues

### Go 模块代理问题
- **问题**: Go proxy.golang.org 连接不稳定，导致 EOF 错误
- **解决方案**: 使用中国镜像代理 `GOPROXY=https://goproxy.cn,direct`
- **命令**: `GOPROXY=https://goproxy.cn,direct GOSUMDB=off go build ./...`

## Requirements
- 深度理解 Entire CLI 项目
- 头脑风暴思考开发方向
- 写入 feature.md
- 调用蜂群技术（planning skill 和 superpower skill）
- 撰写 todo list 并执行
- 中间输出 find.md
- 依次迭代更新循环
- 指导 feature.md 中所有开发方向
- 解决 todo list 中所有事项
- 解决 find.md 所有新发现

## Research Findings

### 项目概述
- **Entire CLI** - Git 工作流工具，通过 hooks 捕获 AI agent 会话
- 将会话数据与 git commit 结合，创建可搜索的代码演变记录
- 支持多种 Agent：Claude Code, Gemini CLI, OpenCode, Cursor, Factory AI Droid, Copilot CLI, Trae Agent

### 已知限制（KNOWN_LIMITATIONS.md）
1. **git commit --amend -m** 会丢失 Entire-Checkpoint trailer
2. **Git GC 可能损坏 worktree 索引**
3. **并发 ACTIVE 会话可能产生虚假检查点**
4. **Cursor rewind 不支持**

### 当前版本功能（CHANGELOG 0.5.1）
- 稀疏元数据获取
- `entire trace` 命令
- 可选 PII 脱敏
- 外部 agent 自动发现
- Checkpoint 远程仓库预览
- E2E 测试改进
- hk hook manager 检测

### Trae Agent 改进计划
- 提取类型定义到 types.go
- 提取生命周期逻辑到 lifecycle.go
- 添加完整测试覆盖
- 添加 AGENT.md 文档

### 开发方向（头脑风暴）
**高优先级：**
- Trae Agent 改进（已在计划中）
- Cursor rewind 支持
- Git GC 损坏自动恢复
- -m 标志 trailer 丢失修复

**中优先级：**
- `entire trace` 命令增强
- 外部 agent 插件系统完善
- 检查点比较功能
- 性能优化

**新 Agent 支持：**
- Windsurf (Codeium)
- Aider (终端 AI 编码助手)
- Devin (Cognition AI)
- Bolt.new (StackBlitz)
- Ollama/LM Studio 集成

**用户体验改进：**
- `entire explain` 增强（AI 生成解释）
- `entire rewind` 增强（交互式差异预览）
- `entire status` 增强（实时会话进度）
- 更好的 CLI 输出

### 技术栈
- Go 1.25.x / 1.26.x
- mise (构建工具)
- golangci-lint
- github.com/spf13/cobra
- github.com/charmbracelet/huh
- github.com/go-git/go-git

## Technical Decisions
| Decision | Rationale |
|----------|-----------|
| 使用 feature.md 记录开发方向 | 便于迭代和版本控制 |
| 使用 planning skill 创建 todo list | Manus 风格的文件规划 |

## Issues Encountered
| Issue | Resolution |
|-------|------------|
| Ralph Loop 启动失败：找不到脚本路径 | 使用正确的完整路径 `/Users/wangxu/.claude/plugins/marketplaces/claude-plugins-official/plugins/ralph-loop/scripts/setup-ralph-loop.sh` |
| 网络访问需要代理 | 设置 `export https_proxy=http://127.0.0.1:7890 http_proxy=http://127.0.0.1:7890 all_proxy=socks5://127.0.0.1:7890` |

## Resources
- 项目文档：CLAUDE.md, README.md, CHANGELOG.md
- 已知限制：docs/KNOWN_LIMITATIONS.md
- 架构文档：docs/architecture/
- Trae Agent 计划：docs/superpowers/plans/2026-03-17-traeagent-improvements.md
- Cursor agent 实现：cmd/entire/cli/agent/cursor/
- Agent 集成清单：docs/architecture/agent-integration-checklist.md
- Agent 实现指南：docs/architecture/agent-guide.md

## Windsurf 研究结果 (2026-03-23)

- **状态**: 阻塞 - Windsurf 二进制未安装
- **结果**: 创建了部分研究文档 `cmd/entire/cli/agent/windsurf/AGENT.md`
- **后续**: 需要用户安装 Windsurf 后继续

## 新 Agent 集成要求
根据 agent-integration-checklist.md，集成新 Agent 需要：

1. **完整转录存储** - 每次检查点存储完整会话转录
2. **原生格式保留** - 以 agent 原生格式存储转录
3. **Hook 事件映射** - 需要映射 TurnStart, TurnEnd, SessionStart, SessionEnd
4. **Rewind/Resume 支持** - 需要实现相关方法
5. **实现 Agent 接口** - 19 个方法需要实现

## Cursor Rewind 状态
- README.md 明确说明 "Rewind is not available at this time"
- Cursor 已经实现了 TranscriptAnalyzer 接口
- TranscriptAnalyzer.GetTranscriptPosition() 已实现（返回行数）
- TranscriptAnalyzer.ExtractModifiedFilesFromOffset() 返回 nil/0（因为 Cursor transcript 不包含 tool_use 信息）
- TranscriptAnalyzer.ExtractPrompts() 和 ExtractSummary() 已实现
- **分析**: rewind 并不完全依赖 TranscriptAnalyzer 获取文件列表
- **GetRewindPoints 流程**:
  1. 查找当前 HEAD 对应的 session
  2. 从 session 的 shadow branch 读取临时检查点
  3. 从 metadata 读取 prompt 信息
- **可能原因**: Cursor hook 事件可能没有正确触发 checkpoint 保存，或者 session 状态没有正确创建
- **结论**: 需要实际测试 Cursor 来调试具体问题。当前技术实现已就绪

## Git GC 损坏问题
- 已知限制文档中记录了此问题
- 症状：`git status` 失败，`invalid sha1 pointer in cache-tree`
- 解决方案：`git read-tree HEAD`
- 需要在 `entire doctor` 中添加自动检测和修复

## -m 标志 trailer 丢失问题

### 深入分析 (2026-03-23)

根据 `docs/KNOWN_LIMITATIONS.md` 的最新说明：

- **实际行为**: Git 对 `-m` 标志传递 `source="message"`（不是 `"commit"`）给 prepare-commit-msg 钩子
- **自动恢复**: 但是 trailer 会在大多数情况下自动恢复（如果 session state 中存在 LastCheckpointID）
- **丢失场景**: 只有在以下情况 trailer 才会丢失：
  1. `-m` 用于新的内容（之前没有 condensation）
  2. `/dev/tty` 不可用（无法交互确认）

**当前恢复机制** (`handleAmendCommitMsg` 函数):
1. 检查 commit message 是否已有 trailer，有则保留
2. 没有则从 session state 中的 LastCheckpointID 恢复
3. LastCheckpointID 在 condensation 完成时设置

**结论**: 大多数情况下已有恢复机制，完全修复的复杂度较高（需要在 commit 本身永久存储 checkpoint ID 或标签）

## Cursor Rewind 状态 (深入分析)

### 技术实现分析

1. **Strategy 层** (`manual_commit_rewind.go`):
   - `GetRewindPoints` 从 shadow branch 读取临时检查点
   - 通过 `findSessionsForCommit` 查找当前 HEAD 对应的 session
   - 从 metadata 读取 prompt 信息

2. **Agent 层** (`cursor/transcript.go`):
   - `GetTranscriptPosition()` - 已实现，返回行数
   - `ExtractPrompts()` - 已实现
   - `ExtractSummary()` - 已实现
   - `ExtractModifiedFilesFromOffset()` - 返回 nil/0（Cursor transcript 不包含 tool_use 信息）

3. **关键发现**:
   - Rewind 并不完全依赖 TranscriptAnalyzer 获取文件列表
   - 文件列表从 shadow branch 的 checkpoint metadata 获取
   - 问题可能在于：Cursor hook 事件没有正确触发 checkpoint 保存

**结论**: 技术实现已就绪，需要实际测试调试来定位具体问题

## Visual/Browser Findings
（本次任务无需浏览器/视觉内容）
