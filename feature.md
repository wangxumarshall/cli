# Entire CLI 开发方向

## 项目概述

Entire 是一个 Git 工作流工具，通过 hooks 捕获 AI agent 会话，并将会话数据与 git commit 结合，创建可搜索的代码演变记录。

## 现有 Agent 支持

- Claude Code
- Gemini CLI
- OpenCode
- Cursor
- Factory AI Droid
- Copilot CLI
- Trae Agent (字节跳动)

---

## 一、已知限制（待解决）

### 1.1 Git 操作相关

#### 1.1.1 使用 `-m` 标志修改提交
- **问题**: `git commit --amend -m "new message"` 会替换整个消息，包括 `Entire-Checkpoint` trailer
- **现状**: 大多数情况下会自动恢复（如果 session state 中存在 LastCheckpointID）
- **改进方向**: 完全解决 -m 标志的 trailer 丢失问题

#### 1.1.2 Git GC 可能损坏 Worktree 索引
- **问题**: `git gc --auto` 可能损坏 worktree 的索引缓存
- **影响**: `git status` 失败，staged 更改丢失
- **恢复**: `git read-tree HEAD`
- **改进方向**:
  - 自动检测并恢复
  - 提供 `entire doctor` 自动修复
  - 配置 git 禁用 auto gc

#### 1.1.3 并发 ACTIVE 会话可能产生虚假检查点
- **问题**: 多个 ACTIVE 会话在同一目录，一个会话的提交会 condensation 所有会话
- **现状**:  cosmetic 问题，无数据丢失
- **改进方向**: 改进会话隔离机制

### 1.2 功能缺失

#### 1.2.1 Cursor Rewind 不支持
- **现状**: Cursor 的 rewind 功能尚未实现
- **改进方向**: 实现 Cursor rewind 支持

### 1.3 性能问题

#### 1.3.1 提交钩子性能
- **现状**: prepare-commit-msg 钩子在大型仓库中性能不佳
- **改进方向**: 继续优化性能

---

## 二、近期功能（CHANGELOG 展示）

### 2.1 0.5.1 版本新增功能

- **稀疏元数据获取**: 按需 blob 解析，减少内存和网络成本
- **`entire trace` 命令**: 诊断慢性能钩子和生命周期事件
- **可选 PII 脱敏**: 带类型标记的敏感信息脱敏
- **外部 agent 自动发现**: 在 `entire enable`, `entire rewind`, `entire resume` 期间自动发现
- **Checkpoint 远程仓库预览**: 支持专用远程仓库存储 checkpoint 数据
- **外部 agent E2E 测试**: Roger Roger canary 测试
- **hk hook manager 检测**

### 2.2 0.5.0 版本新增功能

- **外部 agent 插件系统**: 懒加载发现、超时保护、功能标志门控
- **Vogon 确定性假 agent**: 零成本 E2E canary 测试
- **`entire resume` squash-merge 支持**: 从元数据分支解析 checkpoint ID
- **`entire rewind` squash-merge 支持**
- **模型名称跟踪**: 在会话信息中显示模型名称
- **性能测量**: perf 包，跨所有生命周期钩子的 span 工具

---

## 三、当前计划（已完成 ✅）

### 3.1 Trae Agent 改进计划 ✅ 已完成

参见 `docs/superpowers/plans/2026-03-17-traeagent-improvements.md`

**目标**: 改进 traeagent 实现，对齐其他 agent 的代码结构

**已完成**:
1. ✅ 提取类型定义到 types.go
2. ✅ 提取生命周期逻辑到 lifecycle.go
3. ✅ 添加编译时接口断言
4. ✅ 添加 AGENT.md 文档

**新增文件**:
- `cmd/entire/cli/agent/traeagent/types.go` - 类型定义
- `cmd/entire/cli/agent/traeagent/lifecycle.go` - 生命周期逻辑
- `cmd/entire/cli/agent/traeagent/AGENT.md` - 集成文档

**改进**:
- 添加 HookBeforeAgent 和 HookAfterAgent 支持
- 编译时接口断言确保实现完整性

---

## 四、头脑风暴：未来开发方向

### 4.1 新 Agent 支持

> 调研时间：2026-03-22
> 集成要求：参见 `docs/architecture/agent-integration-checklist.md`

- [ ] **Windsurf** - Codeium 的 AI 代理
- [ ] **Aider** - 终端 AI 编码助手
- [ ] **Devin** - Cognition AI 的软件工程师
- [ ] **Bolt.new** - StackBlitz 的 AI 开发环境
- [ ] **更多本地 LLM 代理**: Ollama, LM Studio 集成

**集成要求：**
1. 实现 Agent 接口（19 个方法）
2. Hook 事件映射（TurnStart, TurnEnd, SessionStart, SessionEnd）
3. 原生格式转录存储
4. Rewind/Resume 支持

### 4.2 用户体验改进

- [ ] **`entire explain` 增强**:
  - AI 生成的代码解释
  - 可视化代码演变图
  - 与 LLM 对话理解代码

- [ ] **`entire rewind` 增强**:
  - 交互式文件差异预览
  - 选择性文件恢复
  - 分支/标签标记检查点

- [ ] **`entire status` 增强**:
  - 实时会话进度
  - Token 使用统计
  - 与其他会话比较

- [ ] **更好的 CLI 输出**:
  - 进度条和动画
  - 颜色主题定制
  - JSON 输出模式（脚本集成）

### 4.3 检查点增强

- [ ] **检查点注释/标签**: 为检查点添加自定义标签便于搜索
- [ ] **检查点比较**: 比较任意两个检查点的差异
- [ ] **检查点导出**: 导出检查点为独立文件/目录
- [ ] **检查点分享**: 生成可分享的检查点链接

### 4.4 搜索和发现

- [ ] **语义搜索**: 基于嵌入的代码语义搜索
- [ ] **时间线视图**: 可视化会话时间线
- [ ] **代码演变图**: 可视化文件随时间的演变

### 4.5 协作功能

- [ ] **团队仪表板**: 查看团队成员的会话
- [ ] **会话模板**: 预定义会话模板
- [ ] **代码审查集成**: 在 PR 中显示相关检查点

### 4.6 集成增强

- [ ] **IDE 插件**:
  - VS Code 插件
  - JetBrains 插件

- [ ] **CI/CD 集成**:
  - GitHub Actions
  - GitLab CI

- [ ] **第三方工具集成**:
  - Linear/Jira 问题关联
  - Slack/Discord 通知

### 4.7 存储和备份

- [ ] **S3/云存储支持**: 将检查点存储到 S3、GCS、Azure Blob
- [ ] **加密存储**: 端到端加密检查点
- [ ] **增量备份**: 高效的增量备份策略

### 4.8 性能优化

- [ ] **并行检查点保存**: 多线程保存检查点
- [ ] **压缩传输**: 压缩 git 操作的数据传输
- [ ] **缓存层**: 本地缓存常用检查点

### 4.9 安全和合规

- [ ] **审计日志**: 详细的操作审计日志
- [ ] **访问控制**: 基于角色的访问控制
- [ ] **合规导出**: GDPR、SOC2 合规导出

### 4.10 开发者体验

- [ ] **调试模式**: 详细的调试输出
- [ ] **Hook 模板**: 自定义 hook 模板
- [ ] **插件系统**: 允许第三方扩展
- [ ] **API**: 允许程序化访问 Entire 功能

---

## 五、优先级建议

### 高优先级（近期）

1. ✅ Trae Agent 改进（已完成）
2. 🔲 Cursor rewind 支持
3. ✅ Git GC 损坏自动恢复（已在 doctor 命令中实现）
4. 🔲 -m 标志 trailer 丢失修复

### 中优先级（中期）

1. 🔲 `entire trace` 命令增强
2. 🔲 外部 agent 插件系统完善
3. 🔲 检查点比较功能
4. 🔲 性能优化

### 低优先级（长期）

1. 🔲 语义搜索
2. 🔲 团队功能
3. 🔲 云存储支持
4. 🔲 IDE 插件

---

## 六、技术债务

- [ ] 测试覆盖提升至 80%+
- [ ] 文档更新（AGENTS.md）
- [ ] 错误消息改进
- [ ] 日志规范化
