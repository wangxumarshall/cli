package codex

import (
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/agent/types"
	"github.com/stretchr/testify/require"
)

// Compile-time interface checks
var (
	_ agent.Agent              = (*CodexAgent)(nil)
	_ agent.HookSupport        = (*CodexAgent)(nil)
	_ agent.HookResponseWriter = (*CodexAgent)(nil)
)

func TestCodexAgent_Name(t *testing.T) {
	t.Parallel()
	ag := NewCodexAgent()
	require.Equal(t, types.AgentName("codex"), ag.Name())
}

func TestCodexAgent_Type(t *testing.T) {
	t.Parallel()
	ag := NewCodexAgent()
	require.Equal(t, types.AgentType("Codex"), ag.Type())
}

func TestCodexAgent_Description(t *testing.T) {
	t.Parallel()
	ag := NewCodexAgent()
	require.Contains(t, ag.Description(), "Codex")
}

func TestCodexAgent_IsPreview(t *testing.T) {
	t.Parallel()
	ag := &CodexAgent{}
	require.True(t, ag.IsPreview())
}

func TestCodexAgent_ProtectedDirs(t *testing.T) {
	t.Parallel()
	ag := &CodexAgent{}
	require.Equal(t, []string{".codex"}, ag.ProtectedDirs())
}

func TestCodexAgent_HookNames(t *testing.T) {
	t.Parallel()
	ag := &CodexAgent{}
	names := ag.HookNames()
	require.Contains(t, names, "session-start")
	require.Contains(t, names, "user-prompt-submit")
	require.Contains(t, names, "stop")
	require.Contains(t, names, "pre-tool-use")
}

func TestCodexAgent_FormatResumeCommand(t *testing.T) {
	t.Parallel()
	ag := &CodexAgent{}
	cmd := ag.FormatResumeCommand("550e8400-e29b-41d4-a716-446655440000")
	require.Equal(t, "codex resume 550e8400-e29b-41d4-a716-446655440000", cmd)
}
