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
