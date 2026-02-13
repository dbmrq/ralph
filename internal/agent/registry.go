package agent

import (
	"fmt"
	"sort"
	"sync"
)

// ErrNoAgentsAvailable is returned when no agents are available.
var ErrNoAgentsAvailable = fmt.Errorf("no agents available")

// ErrAgentNotFound is returned when a requested agent is not found.
var ErrAgentNotFound = fmt.Errorf("agent not found")

// ErrMultipleAgentsAvailable is returned when multiple agents are available
// but no selection was made.
var ErrMultipleAgentsAvailable = fmt.Errorf("multiple agents available, selection required")

// Registry manages registered agent plugins.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]Agent
}

// NewRegistry creates a new agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

// Register adds an agent to the registry.
// If an agent with the same name already exists, it will be replaced.
func (r *Registry) Register(agent Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[agent.Name()] = agent
}

// Unregister removes an agent from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

// Get retrieves an agent by name.
// Returns the agent and true if found, or nil and false if not found.
func (r *Registry) Get(name string) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[name]
	return agent, ok
}

// All returns all registered agents, sorted by name.
func (r *Registry) All() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}

	// Sort by name for consistent ordering
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name() < agents[j].Name()
	})

	return agents
}

// Available returns all agents that are installed and available.
// Agents are sorted by name for consistent ordering.
func (r *Registry) Available() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		if agent.IsAvailable() {
			agents = append(agents, agent)
		}
	}

	// Sort by name for consistent ordering
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name() < agents[j].Name()
	})

	return agents
}

// Names returns the names of all registered agents.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// AvailableCount returns the number of available agents.
func (r *Registry) AvailableCount() int {
	return len(r.Available())
}

// SelectAgent selects an agent based on the given name or availability.
// If name is empty and only one agent is available, it returns that agent.
// If name is empty and multiple agents are available, it returns ErrMultipleAgentsAvailable.
// If name is provided but not found, it returns ErrAgentNotFound.
// If no agents are available, it returns ErrNoAgentsAvailable.
func (r *Registry) SelectAgent(name string) (Agent, error) {
	if name != "" {
		agent, ok := r.Get(name)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrAgentNotFound, name)
		}
		if !agent.IsAvailable() {
			return nil, fmt.Errorf("agent %q is not available", name)
		}
		return agent, nil
	}

	available := r.Available()
	switch len(available) {
	case 0:
		return nil, ErrNoAgentsAvailable
	case 1:
		return available[0], nil
	default:
		return nil, ErrMultipleAgentsAvailable
	}
}

// GetOrDefault returns the agent with the given name, or the single available
// agent if name is empty. This is a convenience method for cases where
// multiple agents should not cause an error.
func (r *Registry) GetOrDefault(name string) (Agent, error) {
	if name != "" {
		agent, ok := r.Get(name)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrAgentNotFound, name)
		}
		return agent, nil
	}

	available := r.Available()
	if len(available) == 0 {
		return nil, ErrNoAgentsAvailable
	}
	// Return first available (sorted by name)
	return available[0], nil
}

// AgentSelector is a function that prompts the user to select an agent.
// It receives a list of available agents and returns the selected agent or an error.
type AgentSelector func(agents []Agent) (Agent, error)

// PromptUserSelection prompts the user to select an agent when multiple are available.
// It uses the provided selector function for the actual UI interaction.
// If only one agent is available, it returns that agent without prompting.
// If no agents are available, it returns ErrNoAgentsAvailable.
func (r *Registry) PromptUserSelection(selector AgentSelector) (Agent, error) {
	available := r.Available()

	switch len(available) {
	case 0:
		return nil, ErrNoAgentsAvailable
	case 1:
		return available[0], nil
	default:
		if selector == nil {
			return nil, ErrMultipleAgentsAvailable
		}
		return selector(available)
	}
}

