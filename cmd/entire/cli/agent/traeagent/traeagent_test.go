// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import (
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/stretchr/testify/assert"
)

func TestNewTraeAgent(t *testing.T) {
	ag := NewTraeAgent()
	assert.NotNil(t, ag)
	assert.Equal(t, agent.AgentNameTraeAgent, ag.Name())
	assert.Equal(t, agent.AgentTypeTraeAgent, ag.Type())
	assert.Equal(t, "Trae Agent - ByteDance's LLM-based software engineering agent", ag.Description())
}

func TestTraeAgent_SupportsHooks(t *testing.T) {
	ag := NewTraeAgent()
	assert.True(t, ag.SupportsHooks())
}

func TestTraeAgent_ProtectedDirs(t *testing.T) {
	ag := NewTraeAgent()
	dirs := ag.ProtectedDirs()
	assert.Equal(t, []string{".trae"}, dirs)
}

func TestTraeAgent_GetHookNames(t *testing.T) {
	ag := NewTraeAgent()
	hookHandler, ok := ag.(agent.HookHandler)
	assert.True(t, ok, "TraeAgent should implement HookHandler")
	hookNames := hookHandler.GetHookNames()
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

func TestTraeAgent_GetSupportedHooks(t *testing.T) {
	ag := NewTraeAgent()
	hookSupport, ok := ag.(agent.HookSupport)
	assert.True(t, ok, "TraeAgent should implement HookSupport")
	supportedHooks := hookSupport.GetSupportedHooks()
	assert.Contains(t, supportedHooks, agent.HookSessionStart)
	assert.Contains(t, supportedHooks, agent.HookSessionEnd)
	assert.Contains(t, supportedHooks, agent.HookBeforeAgent)
	assert.Contains(t, supportedHooks, agent.HookAfterAgent)
	assert.Contains(t, supportedHooks, agent.HookBeforeModel)
	assert.Contains(t, supportedHooks, agent.HookAfterModel)
	assert.Contains(t, supportedHooks, agent.HookBeforeToolSelection)
	assert.Contains(t, supportedHooks, agent.HookPreTool)
	assert.Contains(t, supportedHooks, agent.HookAfterTool)
	assert.Contains(t, supportedHooks, agent.HookPreCompress)
	assert.Contains(t, supportedHooks, agent.HookNotification)
}
