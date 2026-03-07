package iflow

import "encoding/json"

// IFlowSettings represents the .iflow/settings.json structure
type IFlowSettings struct {
	Hooks       IFlowHooks            `json:"hooks,omitempty"`
	Permissions IFlowPermissions      `json:"permissions,omitempty"`
}

// IFlowHooks contains the hook configurations
type IFlowHooks struct {
	PreToolUse       []IFlowHookMatcher `json:"PreToolUse,omitempty"`
	PostToolUse      []IFlowHookMatcher `json:"PostToolUse,omitempty"`
	SetUpEnvironment []IFlowHookEntry   `json:"SetUpEnvironment,omitempty"`
	Stop             []IFlowHookEntry   `json:"Stop,omitempty"`
	SubagentStop     []IFlowHookEntry   `json:"SubagentStop,omitempty"`
	SessionStart     []IFlowHookMatcher `json:"SessionStart,omitempty"`
	SessionEnd       []IFlowHookEntry   `json:"SessionEnd,omitempty"`
	UserPromptSubmit []IFlowHookMatcher `json:"UserPromptSubmit,omitempty"`
	Notification     []IFlowHookMatcher `json:"Notification,omitempty"`
}

// IFlowPermissions contains permission settings
type IFlowPermissions struct {
	Deny []string `json:"deny,omitempty"`
}

// IFlowHookMatcher matches hooks to specific patterns
type IFlowHookMatcher struct {
	Matcher string          `json:"matcher,omitempty"`
	Hooks   []IFlowHookEntry `json:"hooks"`
}

// IFlowHookEntry represents a single hook command
type IFlowHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// --- Hook Input Types ---

// BaseHookInput contains fields common to all hook inputs
type BaseHookInput struct {
	SessionID      string `json:"session_id"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// ToolHookInput is the JSON structure from PreToolUse/PostToolUse hooks
type ToolHookInput struct {
	BaseHookInput
	ToolName     string          `json:"tool_name"`
	ToolAliases  []string        `json:"tool_aliases,omitempty"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
}

// UserPromptSubmitInput is the JSON structure from UserPromptSubmit hook
type UserPromptSubmitInput struct {
	BaseHookInput
	Prompt string `json:"prompt"`
}

// SessionStartInput is the JSON structure from SessionStart hook
type SessionStartInput struct {
	BaseHookInput
	Source string `json:"source,omitempty"` // startup, resume, clear, compress
	Model  string `json:"model,omitempty"`
}

// NotificationInput is the JSON structure from Notification hook
type NotificationInput struct {
	BaseHookInput
	Message string `json:"message"`
}

// StopInput is the JSON structure from Stop hook
type StopInput struct {
	BaseHookInput
	DurationMs int64 `json:"duration_ms,omitempty"`
	TurnCount  int   `json:"turn_count,omitempty"`
}

// SubagentStopInput is the JSON structure from SubagentStop hook
type SubagentStopInput struct {
	BaseHookInput
	SubagentID string `json:"subagent_id,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// --- Transcript Types ---

// TranscriptLine represents a single line in the JSONL transcript
type TranscriptLine struct {
	Type       string          `json:"type"`
	Timestamp  string          `json:"timestamp"`
	Message    json.RawMessage `json:"message,omitempty"`
	ToolUse    *ToolUse        `json:"tool_use,omitempty"`
	ToolResult *ToolResult     `json:"tool_result,omitempty"`
}

// ToolUse represents a tool invocation in the transcript
type ToolUse struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult represents the result of a tool invocation
type ToolResult struct {
	ToolUseID string          `json:"tool_use_id"`
	Result    json.RawMessage `json:"result"`
}

// FileEditToolInput represents input for file editing tools
type FileEditToolInput struct {
	FilePath string `json:"file_path"`
	Path     string `json:"path"` // Alternative field name
}

// FileWriteToolInput represents input for file write tools
type FileWriteToolInput struct {
	FilePath string `json:"file_path"`
	Path     string `json:"path"` // Alternative field name
}

// Tool names used in iFlow CLI transcripts
const (
	ToolWrite        = "write_file"
	ToolEdit         = "replace"
	ToolMultiEdit    = "multi_edit"
	ToolShell        = "run_shell_command"
	ToolRead         = "read_file"
	ToolList         = "list_directory"
	ToolSearch       = "search_file_content"
	ToolGlob         = "glob"
)

// FileModificationTools lists tools that create or modify files
var FileModificationTools = []string{
	ToolWrite,
	ToolEdit,
	ToolMultiEdit,
}

// SessionSource constants for SessionStart hook
type SessionSource string

const (
	SessionSourceStartup  SessionSource = "startup"
	SessionSourceResume   SessionSource = "resume"
	SessionSourceClear    SessionSource = "clear"
	SessionSourceCompress SessionSource = "compress"
)

// IFlow-specific environment variable names
const (
	EnvIFlowSessionID      = "IFLOW_SESSION_ID"
	EnvIFlowTranscriptPath = "IFLOW_TRANSCRIPT_PATH"
	EnvIFlowCWD            = "IFLOW_CWD"
	EnvIFlowHookEventName  = "IFLOW_HOOK_EVENT_NAME"
	EnvIFlowToolName       = "IFLOW_TOOL_NAME"
	EnvIFlowToolArgs       = "IFLOW_TOOL_ARGS"
	EnvIFlowToolAliases    = "IFLOW_TOOL_ALIASES"
	EnvIFlowSessionSource  = "IFLOW_SESSION_SOURCE"
	EnvIFlowUserPrompt     = "IFLOW_USER_PROMPT"
	EnvIFlowNotification   = "IFLOW_NOTIFICATION_MESSAGE"
)

// Settings file name
const IFlowSettingsFileName = "settings.json"

// metadataDenyRule blocks iFlow from reading Entire session metadata
const metadataDenyRule = "Read(./.entire/metadata/**)"

// entireHookPrefixes are command prefixes that identify Entire hooks
var entireHookPrefixes = []string{
	"entire ",
	"go run ${IFLOW_PROJECT_DIR}/cmd/entire/main.go ",
}
