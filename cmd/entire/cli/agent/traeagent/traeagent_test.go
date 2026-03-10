// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import (
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/stretchr/testify/assert"
)

func TestNewTraeAgent(t *testing.T) {
	t.Parallel()
	ag := NewTraeAgent()
	assert.NotNil(t, ag)
	assert.Equal(t, agent.AgentNameTraeAgent, ag.Name())
	assert.Equal(t, agent.AgentTypeTraeAgent, ag.Type())
	assert.Equal(t, "Trae Agent - ByteDance's LLM-based software engineering agent", ag.Description())
}

func TestTraeAgent_HookSupport(t *testing.T) {
	t.Parallel()
	ag := NewTraeAgent()
	_, ok := ag.(agent.HookSupport)
	assert.True(t, ok, "TraeAgent should implement HookSupport")
}

func TestTraeAgent_ProtectedDirs(t *testing.T) {
	t.Parallel()
	ag := NewTraeAgent()
	dirs := ag.ProtectedDirs()
	assert.Equal(t, []string{".trae"}, dirs)
}

func TestTraeAgent_HookNames(t *testing.T) {
	t.Parallel()
	ag := NewTraeAgent()
	hookSupport, ok := ag.(agent.HookSupport)
	assert.True(t, ok, "TraeAgent should implement HookSupport")
	hookNames := hookSupport.HookNames()
	assert.Contains(t, hookNames, HookNameSessionStart)
	assert.Contains(t, hookNames, HookNameSessionEnd)
	assert.Contains(t, hookNames, HookNameBeforeAgent)
	assert.Contains(t, hookNames, HookNameAfterAgent)
	assert.Contains(t, hookNames, HookNameBeforeModel)
	assert.Contains(t, hookNames, HookNameAfterModel)
	assert.Contains(t, hookNames, HookNameBeforeToolSelection)
	assert.Contains(t, hookNames, HookNamePreTool)
	assert.Contains(t, hookNames, HookNameAfterTool)
	assert.Contains(t, hookNames, HookNamePreCompress)
	assert.Contains(t, hookNames, HookNameNotification)
}
