package cmd

import (
	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/agent/auggie"
	"github.com/wexinc/ralph/internal/agent/cursor"
)

// DefaultRegistry is the global registry containing all built-in agents.
// This registry is pre-populated with the Cursor and Auggie agents.
var DefaultRegistry *agent.Registry

// DefaultDiscovery is the global discovery using the DefaultRegistry.
var DefaultDiscovery *agent.Discovery

func init() {
	DefaultRegistry = agent.NewRegistry()
	RegisterDefaultAgents(DefaultRegistry)
	DefaultDiscovery = agent.NewDiscovery(DefaultRegistry)
}

// RegisterDefaultAgents registers all built-in agents with the given registry.
// This is useful for testing when you want a fresh registry with built-in agents.
func RegisterDefaultAgents(r *agent.Registry) {
	r.Register(cursor.New())
	r.Register(auggie.New())
}

