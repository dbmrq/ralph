package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestDiscoveryResult_HasMultiple(t *testing.T) {
	tests := []struct {
		name      string
		available []Agent
		want      bool
	}{
		{"no agents", []Agent{}, false},
		{"single agent", []Agent{&mockAgent{name: "one"}}, false},
		{"two agents", []Agent{&mockAgent{name: "one"}, &mockAgent{name: "two"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DiscoveryResult{Available: tt.available}
			if got := r.HasMultiple(); got != tt.want {
				t.Errorf("HasMultiple() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscoveryResult_HasSingle(t *testing.T) {
	tests := []struct {
		name      string
		available []Agent
		want      bool
	}{
		{"no agents", []Agent{}, false},
		{"single agent", []Agent{&mockAgent{name: "one"}}, true},
		{"two agents", []Agent{&mockAgent{name: "one"}, &mockAgent{name: "two"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DiscoveryResult{Available: tt.available}
			if got := r.HasSingle(); got != tt.want {
				t.Errorf("HasSingle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscoveryResult_HasNone(t *testing.T) {
	tests := []struct {
		name      string
		available []Agent
		want      bool
	}{
		{"no agents", []Agent{}, true},
		{"single agent", []Agent{&mockAgent{name: "one"}}, false},
		{"two agents", []Agent{&mockAgent{name: "one"}, &mockAgent{name: "two"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DiscoveryResult{Available: tt.available}
			if got := r.HasNone(); got != tt.want {
				t.Errorf("HasNone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscovery_Discover(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "available1", available: true, description: "Agent 1"})
	r.Register(&mockAgent{name: "available2", available: true, description: "Agent 2"})
	r.Register(&mockAgent{name: "unavailable", available: false, description: "Not Available"})
	r.Register(&mockAgent{name: "authfail", available: true, authError: errors.New("auth failed")})

	d := NewDiscovery(r)
	result := d.Discover()

	if len(result.Available) != 3 {
		t.Errorf("Available count = %d, want 3", len(result.Available))
	}

	if len(result.Unavailable) != 1 {
		t.Errorf("Unavailable count = %d, want 1", len(result.Unavailable))
	}

	if result.Unavailable[0].Name() != "unavailable" {
		t.Errorf("Unavailable[0].Name() = %q, want %q", result.Unavailable[0].Name(), "unavailable")
	}

	if _, ok := result.AuthErrors["authfail"]; !ok {
		t.Error("Expected auth error for 'authfail' agent")
	}
}

func TestDiscovery_SelectAgent_Configured(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "cursor", available: true})
	r.Register(&mockAgent{name: "auggie", available: true})

	d := NewDiscovery(r)
	agent, err := d.SelectAgent("auggie")

	if err != nil {
		t.Errorf("SelectAgent() error = %v", err)
	}
	if agent.Name() != "auggie" {
		t.Errorf("SelectAgent().Name() = %q, want %q", agent.Name(), "auggie")
	}
}

func TestDiscovery_SelectAgent_SingleAvailable(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "cursor", available: true})

	d := NewDiscovery(r)
	agent, err := d.SelectAgent("")

	if err != nil {
		t.Errorf("SelectAgent() error = %v", err)
	}
	if agent.Name() != "cursor" {
		t.Errorf("SelectAgent().Name() = %q, want %q", agent.Name(), "cursor")
	}
}

func TestDiscovery_SelectAgent_NoAgents(t *testing.T) {
	r := NewRegistry()
	d := NewDiscovery(r)

	_, err := d.SelectAgent("")
	if !errors.Is(err, ErrNoAgentsAvailable) {
		t.Errorf("SelectAgent() error = %v, want %v", err, ErrNoAgentsAvailable)
	}
}

func TestDiscovery_SelectAgent_MultipleWithInput(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "auggie", available: true, description: "Auggie agent"})
	r.Register(&mockAgent{name: "cursor", available: true, description: "Cursor agent"})

	input := strings.NewReader("2\n")
	output := &bytes.Buffer{}

	d := NewDiscovery(r).WithIO(input, output)
	agent, err := d.SelectAgent("")

	if err != nil {
		t.Errorf("SelectAgent() error = %v", err)
	}
	// Agents are sorted alphabetically: auggie=1, cursor=2
	if agent.Name() != "cursor" {
		t.Errorf("SelectAgent().Name() = %q, want %q", agent.Name(), "cursor")
	}

	// Verify prompt was shown
	if !strings.Contains(output.String(), "Multiple AI agents") {
		t.Error("Expected prompt message in output")
	}
}

func TestDiscovery_SelectAgent_InvalidInput(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "auggie", available: true})
	r.Register(&mockAgent{name: "cursor", available: true})

	tests := []struct {
		name  string
		input string
	}{
		{"non-numeric", "abc\n"},
		{"out of range high", "5\n"},
		{"out of range low", "0\n"},
		{"negative", "-1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}

			d := NewDiscovery(r).WithIO(input, output)
			_, err := d.SelectAgent("")

			if err == nil {
				t.Error("SelectAgent() expected error for invalid input")
			}
			if !strings.Contains(err.Error(), "invalid selection") {
				t.Errorf("Error message should contain 'invalid selection', got: %v", err)
			}
		})
	}
}

func TestDiscovery_SelectAgentHeadless(t *testing.T) {
	t.Run("configured agent", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "cursor", available: true})
		r.Register(&mockAgent{name: "auggie", available: true})

		d := NewDiscovery(r)
		agent, err := d.SelectAgentHeadless("cursor")

		if err != nil {
			t.Errorf("SelectAgentHeadless() error = %v", err)
		}
		if agent.Name() != "cursor" {
			t.Errorf("SelectAgentHeadless().Name() = %q, want %q", agent.Name(), "cursor")
		}
	})

	t.Run("single agent", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "cursor", available: true})

		d := NewDiscovery(r)
		agent, err := d.SelectAgentHeadless("")

		if err != nil {
			t.Errorf("SelectAgentHeadless() error = %v", err)
		}
		if agent.Name() != "cursor" {
			t.Errorf("SelectAgentHeadless().Name() = %q, want %q", agent.Name(), "cursor")
		}
	})

	t.Run("multiple agents errors", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "cursor", available: true})
		r.Register(&mockAgent{name: "auggie", available: true})

		d := NewDiscovery(r)
		_, err := d.SelectAgentHeadless("")

		if !errors.Is(err, ErrMultipleAgentsAvailable) {
			t.Errorf("SelectAgentHeadless() error = %v, want %v", err, ErrMultipleAgentsAvailable)
		}
		if !strings.Contains(err.Error(), "--agent flag") {
			t.Error("Error message should mention --agent flag")
		}
	})

	t.Run("no agents", func(t *testing.T) {
		r := NewRegistry()
		d := NewDiscovery(r)

		_, err := d.SelectAgentHeadless("")
		if !errors.Is(err, ErrNoAgentsAvailable) {
			t.Errorf("SelectAgentHeadless() error = %v, want %v", err, ErrNoAgentsAvailable)
		}
	})
}

func TestTerminalSelector(t *testing.T) {
	agents := []Agent{
		&mockAgent{name: "alpha", description: "Alpha agent"},
		&mockAgent{name: "beta", description: "Beta agent"},
	}

	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}

	selector := TerminalSelector(input, output)
	selected, err := selector(agents)

	if err != nil {
		t.Errorf("TerminalSelector() error = %v", err)
	}
	if selected.Name() != "alpha" {
		t.Errorf("TerminalSelector().Name() = %q, want %q", selected.Name(), "alpha")
	}
	if !strings.Contains(output.String(), "alpha - Alpha agent") {
		t.Error("Output should contain agent name and description")
	}
}

func TestDiscovery_DiscoverAndSelect(t *testing.T) {
	t.Run("single agent auto-selects", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "cursor", available: true})

		d := NewDiscovery(r)
		agent, result, err := d.DiscoverAndSelect("", false)

		if err != nil {
			t.Errorf("DiscoverAndSelect() error = %v", err)
		}
		if agent.Name() != "cursor" {
			t.Errorf("Agent.Name() = %q, want %q", agent.Name(), "cursor")
		}
		if !result.HasSingle() {
			t.Error("Result should indicate single agent")
		}
	})

	t.Run("headless mode errors on multiple", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "cursor", available: true})
		r.Register(&mockAgent{name: "auggie", available: true})

		d := NewDiscovery(r)
		_, result, err := d.DiscoverAndSelect("", true)

		if !errors.Is(err, ErrMultipleAgentsAvailable) {
			t.Errorf("DiscoverAndSelect() error = %v, want %v", err, ErrMultipleAgentsAvailable)
		}
		if !result.HasMultiple() {
			t.Error("Result should indicate multiple agents")
		}
	})

	t.Run("configured agent takes precedence", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "cursor", available: true})
		r.Register(&mockAgent{name: "auggie", available: true})

		d := NewDiscovery(r)
		agent, _, err := d.DiscoverAndSelect("auggie", false)

		if err != nil {
			t.Errorf("DiscoverAndSelect() error = %v", err)
		}
		if agent.Name() != "auggie" {
			t.Errorf("Agent.Name() = %q, want %q", agent.Name(), "auggie")
		}
	})
}
