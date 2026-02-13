package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// DiscoveryResult contains the results of agent discovery.
type DiscoveryResult struct {
	// Available is the list of agents that are installed and ready to use.
	Available []Agent
	// Unavailable is the list of agents that are registered but not available.
	Unavailable []Agent
	// AuthErrors maps agent names to authentication errors (if any).
	AuthErrors map[string]error
}

// HasMultiple returns true if more than one agent is available.
func (r *DiscoveryResult) HasMultiple() bool {
	return len(r.Available) > 1
}

// HasSingle returns true if exactly one agent is available.
func (r *DiscoveryResult) HasSingle() bool {
	return len(r.Available) == 1
}

// HasNone returns true if no agents are available.
func (r *DiscoveryResult) HasNone() bool {
	return len(r.Available) == 0
}

// Single returns the single available agent. Panics if not exactly one.
func (r *DiscoveryResult) Single() Agent {
	if len(r.Available) != 1 {
		panic("DiscoveryResult.Single called with != 1 available agents")
	}
	return r.Available[0]
}

// Discovery handles agent detection and selection.
type Discovery struct {
	registry *Registry
	// input is the reader for user input (default: os.Stdin).
	input io.Reader
	// output is the writer for prompts (default: os.Stdout).
	output io.Writer
}

// NewDiscovery creates a new Discovery with the given registry.
func NewDiscovery(registry *Registry) *Discovery {
	return &Discovery{
		registry: registry,
		input:    os.Stdin,
		output:   os.Stdout,
	}
}

// WithIO sets custom input/output for the discovery (for testing).
func (d *Discovery) WithIO(input io.Reader, output io.Writer) *Discovery {
	d.input = input
	d.output = output
	return d
}

// Discover detects all available agents and checks their authentication status.
func (d *Discovery) Discover() *DiscoveryResult {
	result := &DiscoveryResult{
		Available:   make([]Agent, 0),
		Unavailable: make([]Agent, 0),
		AuthErrors:  make(map[string]error),
	}

	for _, agent := range d.registry.All() {
		if agent.IsAvailable() {
			// Check authentication
			if err := agent.CheckAuth(); err != nil {
				result.AuthErrors[agent.Name()] = err
			}
			result.Available = append(result.Available, agent)
		} else {
			result.Unavailable = append(result.Unavailable, agent)
		}
	}

	return result
}

// SelectAgent selects an agent based on configuration and availability.
// If configuredAgent is non-empty, it uses that agent.
// If only one agent is available, it returns that agent.
// If multiple agents are available, it prompts the user to choose.
// If no agents are available, it returns an error.
func (d *Discovery) SelectAgent(configuredAgent string) (Agent, error) {
	// If a specific agent is configured, use it
	if configuredAgent != "" {
		return d.registry.SelectAgent(configuredAgent)
	}

	// Discover available agents
	result := d.Discover()

	if result.HasNone() {
		return nil, ErrNoAgentsAvailable
	}

	if result.HasSingle() {
		return result.Single(), nil
	}

	// Multiple agents available - prompt user to choose
	return d.promptSelection(result.Available)
}

// promptSelection prompts the user to select an agent from the available list.
func (d *Discovery) promptSelection(agents []Agent) (Agent, error) {
	fmt.Fprintln(d.output, "")
	fmt.Fprintln(d.output, "Multiple AI agents are available. Please select one:")
	fmt.Fprintln(d.output, "")

	for i, agent := range agents {
		fmt.Fprintf(d.output, "  [%d] %s - %s\n", i+1, agent.Name(), agent.Description())
	}

	fmt.Fprintln(d.output, "")
	fmt.Fprintf(d.output, "Enter selection (1-%d): ", len(agents))

	scanner := bufio.NewScanner(d.input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read selection: %w", err)
		}
		return nil, fmt.Errorf("no input provided")
	}

	input := strings.TrimSpace(scanner.Text())
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(agents) {
		return nil, fmt.Errorf("invalid selection: %q (expected 1-%d)", input, len(agents))
	}

	return agents[selection-1], nil
}

// SelectAgentHeadless selects an agent for headless mode where user interaction is not possible.
// It returns an error if multiple agents are available and no agent is configured.
// Use the --agent flag or agent.default in config to specify an agent in headless mode.
func (d *Discovery) SelectAgentHeadless(configuredAgent string) (Agent, error) {
	// If a specific agent is configured, use it
	if configuredAgent != "" {
		return d.registry.SelectAgent(configuredAgent)
	}

	// Discover available agents
	result := d.Discover()

	if result.HasNone() {
		return nil, ErrNoAgentsAvailable
	}

	if result.HasSingle() {
		return result.Single(), nil
	}

	// Multiple agents available in headless mode - cannot prompt
	return nil, fmt.Errorf("%w: specify --agent flag or set agent.default in config", ErrMultipleAgentsAvailable)
}

// TerminalSelector returns an AgentSelector function for terminal-based selection.
// This can be passed to Registry.PromptUserSelection for custom selection UI.
func TerminalSelector(input io.Reader, output io.Writer) AgentSelector {
	return func(agents []Agent) (Agent, error) {
		fmt.Fprintln(output, "")
		fmt.Fprintln(output, "Multiple AI agents are available. Please select one:")
		fmt.Fprintln(output, "")

		for i, agent := range agents {
			fmt.Fprintf(output, "  [%d] %s - %s\n", i+1, agent.Name(), agent.Description())
		}

		fmt.Fprintln(output, "")
		fmt.Fprintf(output, "Enter selection (1-%d): ", len(agents))

		scanner := bufio.NewScanner(input)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("failed to read selection: %w", err)
			}
			return nil, fmt.Errorf("no input provided")
		}

		inputStr := strings.TrimSpace(scanner.Text())
		selection, err := strconv.Atoi(inputStr)
		if err != nil || selection < 1 || selection > len(agents) {
			return nil, fmt.Errorf("invalid selection: %q (expected 1-%d)", inputStr, len(agents))
		}

		return agents[selection-1], nil
	}
}

// DefaultTerminalSelector returns a TerminalSelector using os.Stdin and os.Stdout.
func DefaultTerminalSelector() AgentSelector {
	return TerminalSelector(os.Stdin, os.Stdout)
}

// DiscoverAndSelect performs full discovery and selection in one call.
// It detects available agents, checks auth, and either auto-selects the single agent
// or prompts for selection if multiple are available.
// The configuredAgent parameter takes precedence if non-empty.
// Set headless to true when running in CI/headless mode (errors on multiple agents).
func (d *Discovery) DiscoverAndSelect(configuredAgent string, headless bool) (Agent, *DiscoveryResult, error) {
	result := d.Discover()

	// If a specific agent is configured, use it
	if configuredAgent != "" {
		agent, err := d.registry.SelectAgent(configuredAgent)
		return agent, result, err
	}

	if result.HasNone() {
		return nil, result, ErrNoAgentsAvailable
	}

	if result.HasSingle() {
		return result.Single(), result, nil
	}

	// Multiple agents available
	if headless {
		return nil, result, fmt.Errorf("%w: specify --agent flag or set agent.default in config", ErrMultipleAgentsAvailable)
	}

	agent, err := d.promptSelection(result.Available)
	return agent, result, err
}

