// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import "encoding/json"

// Raw data structures for parsing hooks (extracted from traeagent.go)

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

// Additional hook types for HookBeforeAgent and HookAfterAgent
type beforeAgentRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
	Prompt         string `json:"prompt"`
}

type afterAgentRaw struct {
	SessionID      string `json:"session_id"`
	TrajectoryPath string `json:"trajectory_path"`
}

// Hook configuration types (extracted from hooks.go)

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
