package cmd

import (
	"github.com/dbmrq/ralph/internal/agent"
	"github.com/dbmrq/ralph/internal/agent/auggie"
	"github.com/dbmrq/ralph/internal/agent/cursor"
	"github.com/dbmrq/ralph/internal/agent/custom"
	"github.com/dbmrq/ralph/internal/config"
)

// DefaultRegistry is the global registry containing all built-in agents.
// This registry is pre-populated with the Cursor and Auggie agents.
var DefaultRegistry *agent.Registry

// DefaultDiscovery is the global discovery using the DefaultRegistry.
var DefaultDiscovery *agent.Discovery

func init() {
	DefaultRegistry = agent.NewRegistry()
	RegisterDefaultAgents(DefaultRegistry)
	LoadCustomAgents(DefaultRegistry)
	DefaultDiscovery = agent.NewDiscovery(DefaultRegistry)
}

// RegisterDefaultAgents registers all built-in agents with the given registry.
// This is useful for testing when you want a fresh registry with built-in agents.
func RegisterDefaultAgents(r *agent.Registry) {
	r.Register(cursor.New())
	r.Register(auggie.New())
}

// LoadCustomAgents loads custom agents from config and registers them.
// If config cannot be loaded, custom agents are silently skipped.
func LoadCustomAgents(r *agent.Registry) {
	cfg, err := config.Load("")
	if err != nil {
		// Config not found or invalid - silently skip custom agents
		return
	}

	for _, customCfg := range cfg.Agent.Custom {
		a := custom.New(customCfg)
		r.Register(a)
	}
}

// RegisterCustomAgentsFromConfig registers custom agents from a given config.
// This is useful for testing when you want to register agents from a specific config.
func RegisterCustomAgentsFromConfig(r *agent.Registry, cfg *config.Config) {
	for _, customCfg := range cfg.Agent.Custom {
		a := custom.New(customCfg)
		r.Register(a)
	}
}
