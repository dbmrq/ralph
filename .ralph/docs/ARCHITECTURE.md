# Ralph Go - Architecture Document

## Overview

Ralph Go is a complete rewrite of the Ralph Loop shell scripts into a robust Go application with both a TUI (Terminal User Interface) using Bubble Tea and a headless mode for CI/automation. The goal is to create a more maintainable, testable, and feature-rich tool for automated AI agent task execution.

**Note**: This is a clean rewrite. The existing shell scripts are for inspiration only - no backward compatibility is required. When the Go version is complete, shell scripts will be removed.

## Core Design Principles

1. **Clean Architecture**: Separate concerns into distinct layers (UI, business logic, infrastructure)
2. **Agent Plugin System**: Agents are plugins, not hardcoded - easy to add new agents via interface
3. **Dual Interface**: TUI and headless modes are equally supported from the start
4. **Extensible Hooks**: Pre/post task actions (shell commands or agent calls) with configurable failure behavior
5. **Self-Improving Loop**: Agents update documentation and task files directly with learnings
6. **Smart Sub-tasking**: Agents can spawn sub-agents for parallelization when appropriate

## Package Structure

```
ralph/
├── cmd/
│   └── ralph/
│       └── main.go           # Entry point
├── internal/
│   ├── app/
│   │   └── app.go            # Application orchestration
│   ├── agent/
│   │   ├── interface.go      # Agent interface & registry
│   │   ├── registry.go       # Agent plugin registry
│   │   ├── cursor/           # Cursor agent plugin
│   │   │   └── cursor.go
│   │   └── auggie/           # Auggie agent plugin
│   │       └── auggie.go
│   ├── config/
│   │   ├── config.go         # Configuration management
│   │   └── loader.go         # Config file loading (YAML only)
│   ├── task/
│   │   ├── task.go           # Task model
│   │   ├── store.go          # JSON task storage
│   │   └── manager.go        # Task state management
│   ├── prompt/
│   │   ├── builder.go        # Prompt construction
│   │   └── templates.go      # Prompt templates
│   ├── hooks/
│   │   ├── hook.go           # Hook interface
│   │   ├── shell.go          # Shell command hooks
│   │   └── agent.go          # Agent call hooks
│   ├── loop/
│   │   ├── loop.go           # Main loop logic
│   │   ├── session.go        # Session management (pause/resume)
│   │   └── state.go          # Loop state machine
│   ├── git/
│   │   └── git.go            # Git operations
│   ├── build/
│   │   └── build.go          # Build/test verification
│   ├── tui/
│   │   ├── app.go            # TUI application
│   │   ├── model.go          # Bubble Tea model
│   │   ├── view.go           # View rendering
│   │   ├── update.go         # Event handling
│   │   ├── components/       # Reusable UI components
│   │   │   ├── progress.go   # Progress bar
│   │   │   ├── taskList.go   # Task list view
│   │   │   ├── taskEditor.go # Task add/edit interface
│   │   │   ├── log.go        # Log viewport
│   │   │   ├── modelPicker.go # Model selection
│   │   │   └── status.go     # Status bar
│   │   └── styles/
│   │       └── styles.go     # Lipgloss styles
│   └── headless/
│       └── runner.go         # Headless execution mode
├── pkg/
│   └── models/
│       └── models.go         # Shared data models
├── go.mod
├── go.sum
└── README.md
```

## Agent Plugin System

Agents are implemented as plugins that register with a central registry. This makes adding new agents straightforward.

```go
// Agent interface - all agents must implement this
type Agent interface {
    // Identity
    Name() string
    Description() string

    // Availability
    IsAvailable() bool
    CheckAuth() error

    // Models
    ListModels() ([]Model, error)
    GetDefaultModel() Model

    // Execution
    Run(ctx context.Context, prompt string, opts RunOptions) (Result, error)

    // Session management (for pause/resume)
    GetSessionID() string
    Continue(ctx context.Context, sessionID string, prompt string, opts RunOptions) (Result, error)
}

// Registry for agent plugins
type Registry struct {
    agents map[string]Agent
}

func (r *Registry) Register(agent Agent)
func (r *Registry) Get(name string) (Agent, bool)
func (r *Registry) Available() []Agent
func (r *Registry) PromptUserSelection() (Agent, error)  // When multiple available

type RunOptions struct {
    Model         string
    WorkDir       string
    LogPath       string
    LogWriter     io.Writer     // For real-time output streaming
    Timeout       time.Duration // Smart timeout (default 2h if active, 30min if stuck)
    Force         bool
}

type Result struct {
    Output    string
    ExitCode  int
    Duration  time.Duration
    Status    TaskStatus // NEXT, DONE, ERROR, FIXED
    SessionID string     // For continuing paused tasks
}
```

### Adding a New Agent

New agents can be added via a CLI command that prompts for required information:

```bash
ralph agent add
# Prompts for: name, command, detection method, model listing command, etc.
```

## Task Storage

Tasks are stored in JSON format internally (`.ralph/tasks.json`), with a nice TUI interface for viewing and editing. This allows flexible task formats while maintaining structure.

```go
type Task struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Status      TaskStatus        `json:"status"`      // pending, in_progress, completed, skipped, paused, failed
    Order       int               `json:"order"`       // Execution order
    SessionID   string            `json:"session_id"`  // For resuming paused tasks
    Iterations  int               `json:"iterations"`  // Attempts on this task
    Metadata    map[string]string `json:"metadata"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type TaskStore struct {
    Tasks []Task `json:"tasks"`
}
```

## TUI Features

- **Header**: Project name, agent, model, run ID
- **Task Progress**: Visual progress bar with completed/remaining counts
- **Task List**: Scrollable list with status icons, add/edit/reorder capabilities
- **Task Editor**: In-TUI task creation and editing
- **Model Picker**: Select model at startup, mid-run, or via config
- **Agent Output**: Real-time streaming with scrollable overlay
- **Log Viewer**: Scrollable overlay + option to open in $EDITOR, shows file path
- **Status Bar**: Elapsed time, iteration count, build/test status
- **Keyboard Shortcuts**: Pause (p), skip (s), abort (a), logs (l), help (h), review (r)

## Headless Mode

Equally supported for CI/GitHub Actions:

```bash
ralph run --headless
ralph run --headless --output json  # Structured output
```

## Hook System

```go
type Hook interface {
    Name() string
    Type() HookType // Pre or Post
    Execute(ctx context.Context, task *Task, result *Result) error
}

type HookConfig struct {
    PreTask  []HookDefinition
    PostTask []HookDefinition
}

type HookDefinition struct {
    Type      string       // "shell" or "agent"
    Command   string       // Shell command or agent prompt
    Model     string       // For agent hooks (optional, defaults to main agent)
    Agent     string       // For agent hooks (optional, defaults to main agent)
    OnFailure FailureMode  // skip_task, warn_continue, abort_loop, ask_agent
}

type FailureMode string

const (
    FailureSkipTask    FailureMode = "skip_task"
    FailureWarnContinue FailureMode = "warn_continue"
    FailureAbortLoop   FailureMode = "abort_loop"
    FailureAskAgent    FailureMode = "ask_agent"  // Let agent decide
)
```

## TDD Support

The system distinguishes between gate tests and TDD tests with a global baseline (configurable).

```go
type TestMode string

const (
    TestModeGate   TestMode = "gate"   // Block on failure
    TestModeTDD    TestMode = "tdd"    // Allow initial failures, block on regression
    TestModeReport TestMode = "report" // Report only, don't block
)

type TestConfig struct {
    Mode          TestMode `yaml:"mode"`
    Command       string   `yaml:"command"`        // Test command (AI-detected if empty)
    BaselineFile  string   `yaml:"baseline_file"`  // Default: .ralph/test_baseline.json
    BaselineScope string   `yaml:"baseline_scope"` // global (default), session, task
}
```

Test results are captured via exit codes (works for most languages). Custom parsing can be configured when needed.

### Bootstrap-Aware Verification

The verification system gracefully handles greenfield projects using AI-driven analysis:

1. **Project Analysis**: AI agent detects if project has buildable code/tests
2. **Graceful Skip**: If not ready, skip verification with info message (exit 0)
3. **Transition Handling**: When first code/tests appear, capture baseline automatically
4. **Clear Logging**: Always explain why verification was skipped or what was checked

The `ProjectAnalysis.Build.Ready` and `ProjectAnalysis.Test.Ready` fields from the analysis agent determine bootstrap state.

## Feedback System

Instead of a separate feedback storage system, agents are instructed to update existing files directly:

- Update documentation with learnings
- Update task descriptions with context
- Add notes to relevant files

This keeps learnings in context where they're most useful and avoids stale feedback accumulation.

## Smart Timeout System

```go
type TimeoutConfig struct {
    ActiveTimeout time.Duration `yaml:"active_timeout"` // Default: 2h (when agent is producing output)
    StuckTimeout  time.Duration `yaml:"stuck_timeout"`  // Default: 30min (no output change)
}
```

The system monitors agent output to determine if it's actively working or stuck.

## Configuration

All configuration via YAML (`.ralph/config.yaml`):

```yaml
agent:
  default: ""  # Empty = prompt if multiple available

timeout:
  active: 2h
  stuck: 30m

git:
  auto_commit: true  # After each successful task
  commit_prefix: "[ralph]"

build:
  command: ""              # Build command (AI-detected if empty)

test:
  command: ""              # Test command (AI-detected if empty)
  mode: gate               # gate | tdd | report
  baseline_file: .ralph/test_baseline.json
  baseline_scope: global   # global | session | task

hooks:
  pre_task: []
  post_task: []
```

### AI-Driven Project Analysis

Instead of hardcoded language-specific patterns, Ralph uses the AI agent itself to analyze the project. Before starting the task loop, a **Project Analysis Agent** runs once per session to detect:

- Project type and languages
- Build and test commands
- Greenfield/bootstrap state
- Dependency manager and status
- Project context for enhanced prompts

This approach is:
- **Language-agnostic**: Works with any language, including future ones
- **Zero configuration**: No user setup required
- **Context-aware**: AI understands project context, not just file patterns
- **Self-improving**: As AI models improve, detection improves automatically

See `FEATURES.md` Section 4 for full specification.

