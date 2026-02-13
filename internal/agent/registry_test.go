package agent

import (
	"errors"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("NewRegistry().Count() = %d, want 0", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	agent := &mockAgent{name: "test", available: true}

	r.Register(agent)

	if r.Count() != 1 {
		t.Errorf("Count() = %d, want 1", r.Count())
	}

	got, ok := r.Get("test")
	if !ok {
		t.Error("Get() returned false, want true")
	}
	if got.Name() != "test" {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), "test")
	}
}

func TestRegistry_Register_Replaces(t *testing.T) {
	r := NewRegistry()
	agent1 := &mockAgent{name: "test", description: "first"}
	agent2 := &mockAgent{name: "test", description: "second"}

	r.Register(agent1)
	r.Register(agent2)

	if r.Count() != 1 {
		t.Errorf("Count() = %d, want 1", r.Count())
	}

	got, _ := r.Get("test")
	if got.Description() != "second" {
		t.Errorf("Description() = %q, want %q", got.Description(), "second")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "test", available: true})

	r.Unregister("test")

	if r.Count() != 0 {
		t.Errorf("Count() = %d, want 0", r.Count())
	}

	_, ok := r.Get("test")
	if ok {
		t.Error("Get() returned true after Unregister")
	}
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "charlie", available: true})
	r.Register(&mockAgent{name: "alpha", available: false})
	r.Register(&mockAgent{name: "bravo", available: true})

	all := r.All()

	if len(all) != 3 {
		t.Errorf("All() returned %d agents, want 3", len(all))
	}

	// Verify sorted order
	expected := []string{"alpha", "bravo", "charlie"}
	for i, name := range expected {
		if all[i].Name() != name {
			t.Errorf("All()[%d].Name() = %q, want %q", i, all[i].Name(), name)
		}
	}
}

func TestRegistry_Available(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "unavailable", available: false})
	r.Register(&mockAgent{name: "available1", available: true})
	r.Register(&mockAgent{name: "available2", available: true})

	available := r.Available()

	if len(available) != 2 {
		t.Errorf("Available() returned %d agents, want 2", len(available))
	}

	// Verify sorted order
	if available[0].Name() != "available1" {
		t.Errorf("Available()[0].Name() = %q, want %q", available[0].Name(), "available1")
	}
	if available[1].Name() != "available2" {
		t.Errorf("Available()[1].Name() = %q, want %q", available[1].Name(), "available2")
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "charlie"})
	r.Register(&mockAgent{name: "alpha"})
	r.Register(&mockAgent{name: "bravo"})

	names := r.Names()

	expected := []string{"alpha", "bravo", "charlie"}
	if len(names) != len(expected) {
		t.Fatalf("Names() returned %d names, want %d", len(names), len(expected))
	}

	for i, name := range expected {
		if names[i] != name {
			t.Errorf("Names()[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestRegistry_AvailableCount(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAgent{name: "one", available: true})
	r.Register(&mockAgent{name: "two", available: false})
	r.Register(&mockAgent{name: "three", available: true})

	if got := r.AvailableCount(); got != 2 {
		t.Errorf("AvailableCount() = %d, want 2", got)
	}
}

func TestRegistry_SelectAgent(t *testing.T) {
	tests := []struct {
		name      string
		agents    []*mockAgent
		selection string
		wantName  string
		wantErr   error
	}{
		{
			name:    "no agents",
			agents:  []*mockAgent{},
			wantErr: ErrNoAgentsAvailable,
		},
		{
			name:      "single available - no selection",
			agents:    []*mockAgent{{name: "only", available: true}},
			selection: "",
			wantName:  "only",
		},
		{
			name: "multiple available - no selection",
			agents: []*mockAgent{
				{name: "one", available: true},
				{name: "two", available: true},
			},
			selection: "",
			wantErr:   ErrMultipleAgentsAvailable,
		},
		{
			name: "select by name",
			agents: []*mockAgent{
				{name: "one", available: true},
				{name: "two", available: true},
			},
			selection: "two",
			wantName:  "two",
		},
		{
			name:      "select non-existent",
			agents:    []*mockAgent{{name: "one", available: true}},
			selection: "missing",
			wantErr:   ErrAgentNotFound,
		},
		{
			name:      "select unavailable",
			agents:    []*mockAgent{{name: "test", available: false}},
			selection: "test",
			wantErr:   nil, // Error is not a sentinel error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			for _, a := range tt.agents {
				r.Register(a)
			}

			got, err := r.SelectAgent(tt.selection)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("SelectAgent() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// Special case for unavailable agent error
			if tt.name == "select unavailable" {
				if err == nil {
					t.Error("SelectAgent() expected error for unavailable agent")
				}
				return
			}

			if err != nil {
				t.Errorf("SelectAgent() unexpected error = %v", err)
				return
			}

			if got.Name() != tt.wantName {
				t.Errorf("SelectAgent().Name() = %q, want %q", got.Name(), tt.wantName)
			}
		})
	}
}

func TestRegistry_GetOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		agents   []*mockAgent
		lookup   string
		wantName string
		wantErr  error
	}{
		{
			name:    "no agents",
			agents:  []*mockAgent{},
			lookup:  "",
			wantErr: ErrNoAgentsAvailable,
		},
		{
			name:     "single agent - no name",
			agents:   []*mockAgent{{name: "only", available: true}},
			lookup:   "",
			wantName: "only",
		},
		{
			name: "multiple agents - no name returns first",
			agents: []*mockAgent{
				{name: "bravo", available: true},
				{name: "alpha", available: true},
			},
			lookup:   "",
			wantName: "alpha", // First alphabetically
		},
		{
			name:     "get by name",
			agents:   []*mockAgent{{name: "test", available: true}},
			lookup:   "test",
			wantName: "test",
		},
		{
			name:    "get missing by name",
			agents:  []*mockAgent{{name: "test", available: true}},
			lookup:  "missing",
			wantErr: ErrAgentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			for _, a := range tt.agents {
				r.Register(a)
			}

			got, err := r.GetOrDefault(tt.lookup)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetOrDefault() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetOrDefault() unexpected error = %v", err)
				return
			}

			if got.Name() != tt.wantName {
				t.Errorf("GetOrDefault().Name() = %q, want %q", got.Name(), tt.wantName)
			}
		})
	}
}

func TestRegistry_PromptUserSelection(t *testing.T) {
	t.Run("no agents", func(t *testing.T) {
		r := NewRegistry()
		_, err := r.PromptUserSelection(nil)
		if !errors.Is(err, ErrNoAgentsAvailable) {
			t.Errorf("PromptUserSelection() error = %v, want %v", err, ErrNoAgentsAvailable)
		}
	})

	t.Run("single agent - no prompt", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "only", available: true})

		// Selector should not be called with single agent
		called := false
		selector := func(_ []Agent) (Agent, error) {
			called = true
			return nil, nil
		}

		got, err := r.PromptUserSelection(selector)
		if err != nil {
			t.Errorf("PromptUserSelection() error = %v", err)
		}
		if got.Name() != "only" {
			t.Errorf("PromptUserSelection().Name() = %q, want %q", got.Name(), "only")
		}
		if called {
			t.Error("selector was called for single agent")
		}
	})

	t.Run("multiple agents - uses selector", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "alpha", available: true})
		r.Register(&mockAgent{name: "bravo", available: true})

		// Selector returns the second agent
		selector := func(agents []Agent) (Agent, error) {
			if len(agents) != 2 {
				t.Errorf("selector received %d agents, want 2", len(agents))
			}
			return agents[1], nil
		}

		got, err := r.PromptUserSelection(selector)
		if err != nil {
			t.Errorf("PromptUserSelection() error = %v", err)
		}
		// "bravo" is second alphabetically
		if got.Name() != "bravo" {
			t.Errorf("PromptUserSelection().Name() = %q, want %q", got.Name(), "bravo")
		}
	})

	t.Run("multiple agents - nil selector returns error", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "alpha", available: true})
		r.Register(&mockAgent{name: "bravo", available: true})

		_, err := r.PromptUserSelection(nil)
		if !errors.Is(err, ErrMultipleAgentsAvailable) {
			t.Errorf("PromptUserSelection() error = %v, want %v", err, ErrMultipleAgentsAvailable)
		}
	})

	t.Run("selector error propagates", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&mockAgent{name: "alpha", available: true})
		r.Register(&mockAgent{name: "bravo", available: true})

		expectedErr := errors.New("user cancelled")
		selector := func(_ []Agent) (Agent, error) {
			return nil, expectedErr
		}

		_, err := r.PromptUserSelection(selector)
		if !errors.Is(err, expectedErr) {
			t.Errorf("PromptUserSelection() error = %v, want %v", err, expectedErr)
		}
	})
}

