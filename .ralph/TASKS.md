# Ralph Go Rewrite - Task List

**Purpose:** Transform Ralph Loop from shell scripts to a robust Go application with TUI
**Reference:** See `.ralph/docs/` for architecture and feature specifications

**Note:**  If during development you realize there are architecture improvements that can be made to subsequent tasks, update them!

---

## 🏗️ Phase 1: Project Foundation

> **Note**: These are setup-only tasks. No tests are needed for INIT-001 and INIT-002.
> INIT-003 should include basic tests for the CLI commands.

- [x] INIT-001: Initialize Go module and basic project structure
  > Goal: Create go.mod, directory structure per ARCHITECTURE.md
  > Create cmd/ralph/main.go with minimal entry point
  > Add .gitignore for Go projects
  > Tests: Not required (setup-only task)

- [x] INIT-002: Add core dependencies
  > Goal: Add Bubble Tea, Lip Gloss, Cobra, Viper dependencies
  > github.com/charmbracelet/bubbletea
  > github.com/charmbracelet/lipgloss
  > github.com/spf13/cobra
  > github.com/spf13/viper
  > gopkg.in/yaml.v3
  > Tests: Not required (dependency-only task)

- [x] INIT-003: Create basic CLI structure with Cobra
  > Goal: Implement root command with version flag
  > Add `run` subcommand skeleton
  > Add `init` subcommand skeleton for project setup
  > Add `--headless` flag from the start
  > Tests: Add basic CLI tests (command parsing, flag handling)

---

## 🔧 Phase 2: Configuration System

- [x] CONFIG-001: Define configuration data structures
  > Goal: Create config.go with Config struct matching all settings
  > Include agent settings, loop settings, git settings, gates, hooks
  > YAML-only format (no shell script support needed)

- [x] CONFIG-002: Implement configuration loader
  > Goal: Load config from .ralph/config.yaml
  > Support environment variable overrides
  > Validate configuration and provide clear error messages

- [x] CONFIG-003: Add prompt template loading
  > Goal: Load base_prompt.txt, platform_prompt.txt, project_prompt.txt
  > Implement 3-level prompt system from ARCHITECTURE.md
  > Support template variables substitution

- [x] CONFIG-004: Add smart timeout configuration
  > Goal: Implement dual timeout system (active vs stuck)
  > Default: active=2h, stuck=30m
  > Monitor agent output to detect stuck state

---

## 📋 Phase 3: Task Management

- [x] TASK-001: Create task data model
  > Goal: Define Task struct with ID, name, description, status, metadata
  > Support task states: pending, in_progress, completed, skipped, paused, failed
  > Include iteration history per task

- [x] TASK-002: Implement JSON task storage
  > Goal: Store tasks internally as JSON (.ralph/tasks.json)
  > Define schema: id, name, description, status, created_at, completed_at
  > Support import from various text formats on init

- [x] TASK-003: Implement task manager
  > Goal: Create TaskManager for state management
  > Methods: GetNext(), MarkComplete(), Skip(), Pause(), CountRemaining()
  > Handle task ordering and dependencies

- [x] TASK-004: Add task import utilities
  > Goal: Parse markdown task lists on import
  > Parse plain text task lists
  > Convert to internal JSON format

---

## 🤖 Phase 4: Agent Plugin System

- [x] AGENT-001: Define Agent interface and Registry
  > Goal: Create agent.go with Agent interface per ARCHITECTURE.md
  > Define Model, RunOptions, Result types with SessionID field
  > Implement AgentRegistry for plugin management

- [x] AGENT-002: Implement Cursor agent plugin
  > Goal: Create cursor.go implementing Agent interface
  > Register with AgentRegistry on init
  > Detect via `agent` command availability
  > Parse `agent --list-models` for model discovery
  > Execute with flags: --print, --force, --model

- [ ] AGENT-003: Implement Auggie agent plugin
  > Goal: Create auggie.go implementing Agent interface per AUGGIE_INTEGRATION.md
  > Register with AgentRegistry on init
  > Handle session token authentication (AUGMENT_SESSION_AUTH)
  > Execute with --print --quiet, support --continue for session resumption
  > Parse `auggie models list` for model discovery

- [ ] AGENT-004: Add agent discovery and selection
  > Goal: On startup, detect all available agents
  > If multiple available, prompt user to choose (no silent defaults)
  > Store selection in session state

- [ ] AGENT-005: Add agent add command
  > Goal: `ralph agent add` for adding custom agents
  > Prompt for: name, command, detection method, model list command
  > Store in config for future sessions

---

## 🔗 Phase 5: Hook System

- [ ] HOOK-001: Define hook interface and types
  > Goal: Create hook.go with Hook interface
  > Define HookType (Pre/Post), HookConfig, HookDefinition
  > Support shell and agent hook types
  > Define failure modes: skip_task, warn_continue, abort_loop, ask_agent

- [ ] HOOK-002: Implement shell command hooks
  > Goal: Create shell.go for shell command execution
  > Set environment variables (TASK_ID, TASK_NAME, TASK_STATUS, etc.)
  > Capture output and handle errors per on_failure mode

- [ ] HOOK-003: Implement agent call hooks
  > Goal: Create agenthook.go for agent-based hooks
  > Run agent with custom prompt
  > Support optional model and agent specification

- [ ] HOOK-004: Add hook manager
  > Goal: Create HookManager for hook orchestration
  > Execute pre-task and post-task hooks in order
  > Handle failures according to configured failure modes

---

## 🔨 Phase 6: Build & Test System

- [ ] BUILD-001: Implement bootstrap/greenfield detection
  > Goal: Create bootstrap.go with project state detection
  > Auto-detect based on project type (Go: go.mod + *.go, Node: package.json, etc.)
  > Support manual mode with custom bootstrap_check command
  > Separate detection for build-ready vs test-ready states
  > Return BootstrapState: { BuildReady, TestReady, Reason }
  > Reference: FEATURES.md Section 4 "Bootstrap/Greenfield Detection"

- [ ] BUILD-002: Implement build verification
  > Goal: Create build.go with build execution logic
  > Check bootstrap state first - skip gracefully if not ready
  > Support custom build commands from config
  > Auto-detect build command if not configured (go build, npm run build, etc.)
  > Parse build output for errors
  > Return structured BuildResult with bootstrap awareness

- [ ] BUILD-003: Implement test verification
  > Goal: Add test execution with configurable commands
  > Check bootstrap state first - skip gracefully if no test files
  > Primary: Use exit codes for pass/fail
  > Optional: Custom parsing for detailed test names
  > Extract passing/failing test counts
  > Return structured TestResult with bootstrap awareness

- [ ] BUILD-004: Add TDD mode support
  > Goal: Implement test baseline capture and comparison
  > Handle bootstrap phase: no baseline until tests exist
  > Auto-capture baseline when first test file appears
  > Global baseline by default (captured once at start)
  > Track newly passing vs regressing tests
  > Store baseline in .ralph/test_baseline.json
  > Block only on regressions, not pre-existing failures
  > Log clear messages: "No tests yet", "Baseline captured", "N regressions detected"
  > Reference: FEATURES.md Section 5 "TDD Mode"

- [ ] BUILD-005: Create verification gate logic
  > Goal: Orchestrate bootstrap check → build → test → gate decision
  > Support gate, tdd, and report modes
  > Handle transitions: bootstrap → ready (log and capture baseline)
  > Parse task metadata for gate overrides (Tests: Not required, Build: Not required)
  > Skip gates when task metadata says not required (log info message)
  > Clear error messaging for failures
  > Return GateResult: { Passed, Skipped (bootstrap), SkippedByTask, Failed, Reason }

---

## 🔄 Phase 7: Main Loop & Session Management

- [ ] LOOP-001: Create loop state machine
  > Goal: Define LoopState enum and transitions
  > States: Idle, Running, Paused, AwaitingFix, Completed, Failed
  > Handle state persistence for resume

- [ ] LOOP-002: Implement core loop execution
  > Goal: Create loop.go with main execution logic
  > Integrate: task selection → hooks → agent → verify → commit
  > Respect iteration limits per task

- [ ] LOOP-003: Add automatic commit logic
  > Goal: Create git.go for git operations
  > Stage changes, create commit with task reference
  > Handle uncommitted changes detection
  > Configurable commit prefix (default: "[ralph]")

- [ ] LOOP-004: Implement error recovery
  > Goal: Handle agent failures with retry logic
  > Automatic fix attempts for build/test failures
  > Save state on interruption for resume

- [ ] LOOP-005: Implement session management
  > Goal: Generate unique session IDs
  > Persist session state to .ralph/sessions/<id>.json
  > Support `ralph run --continue` for resuming sessions
  > Store agent session IDs for `auggie --continue`

- [ ] LOOP-006: Add headless runner
  > Goal: Implement headless execution mode
  > Support --output json for structured output
  > Same functionality as TUI mode without interactive UI
  > Suitable for CI/GitHub Actions

---

## 🖥️ Phase 8: TUI Implementation

- [ ] TUI-001: Create basic Bubble Tea app structure
  > Goal: Create tui/app.go with Model, Init, Update, View
  > Define messages for state updates
  > Basic key handling (q to quit)

- [ ] TUI-002: Implement header component
  > Goal: Create components/header.go
  > Display project name, agent, model, session ID
  > Style with Lip Gloss

- [ ] TUI-003: Implement progress bar component
  > Goal: Create components/progress.go
  > Show completed/remaining task counts
  > Visual progress indicator with iteration count

- [ ] TUI-004: Implement task list component
  > Goal: Create components/taskList.go
  > Scrollable task list with status icons (✓ ○ → ⊘ ⏸ ✗)
  > Highlight current task
  > Support j/k or arrow key navigation

- [ ] TUI-005: Implement log viewport component
  > Goal: Create components/log.go
  > Real-time agent output streaming
  > Scrollable with auto-follow
  > Option to open in $EDITOR

- [ ] TUI-006: Implement status bar component
  > Goal: Create components/status.go
  > Show elapsed time, iteration count, build/test status
  > Display keyboard shortcuts

- [ ] TUI-007: Implement task editor component
  > Goal: Create components/taskEditor.go
  > Add new tasks inline (e key)
  > Edit task name/description
  > Reorder tasks
  > Save to JSON storage

- [ ] TUI-008: Implement model picker component
  > Goal: Create components/modelPicker.go
  > List available models from current agent
  > Allow selection before or during run (m key)
  > Show current model indicator

- [ ] TUI-009: Add keyboard controls
  > Goal: Handle p (pause), s (skip), a (abort), l (logs), h (help)
  > e (edit tasks), m (model picker), r (review mode)
  > Confirmation dialogs for destructive actions
  > Help overlay

- [ ] TUI-010: Integrate TUI with loop execution
  > Goal: Connect loop events to TUI updates
  > Real-time progress updates
  > Error state visualization
  > Pause/resume integration with session management

---

## 🧠 Phase 9: Agent Instructions

- [ ] INSTR-001: Create prompt builder
  > Goal: Build agent prompts from template layers
  > Combine base_prompt + platform_prompt + project_prompt + task
  > Include relevant context from docs and previous changes

- [ ] INSTR-002: Add plan evolution instructions
  > Goal: Instruct agents to update docs/tasks when plans change
  > "Update remaining tasks if implementation changes the plan"
  > "Document patterns and learnings in project files"

---

## 🧪 Phase 10: Comprehensive Testing

> **Note**: Unit tests are written alongside each task (per base_prompt.txt Phase 4).
> This phase adds **comprehensive integration tests** and improves coverage.

- [ ] TEST-001: Add unit tests for configuration
  > Goal: Test config loading, validation, defaults
  > Test environment variable overrides
  > Test timeout configuration

- [ ] TEST-002: Add unit tests for task management
  > Goal: Test JSON storage read/write
  > Test task status updates
  > Test task import from various formats

- [ ] TEST-003: Add unit tests for agents
  > Goal: Mock agent execution
  > Test AgentRegistry plugin system
  > Test model listing, error handling
  > Test session continuation

- [ ] TEST-004: Add unit tests for hooks
  > Goal: Test hook execution
  > Test failure modes (skip_task, warn_continue, etc.)
  > Test environment variable injection

- [ ] TEST-005: Add integration tests
  > Goal: Test full loop with mock agent
  > Test session persistence and resume
  > Test headless mode output

- [ ] TEST-006: Add TUI tests
  > Goal: Test TUI state updates
  > Test keyboard handling
  > Use Bubble Tea testing utilities

---

## 📦 Phase 11: Polish & Documentation

- [ ] POLISH-001: Add init command implementation
  > Goal: `ralph init` creates .ralph directory
  > Generate default config.yaml
  > Create empty tasks.json
  > Copy prompt templates

- [ ] POLISH-002: Add comprehensive error messages
  > Goal: Clear, actionable error messages
  > Suggest fixes for common issues (auth, missing config, etc.)
  > Include relevant documentation links

- [ ] POLISH-003: Add logging system
  > Goal: Structured logging to .ralph/logs/
  > Log rotation and cleanup
  > Debug, info, error levels

- [ ] POLISH-004: Performance optimization
  > Goal: Profile and optimize hot paths
  > Efficient output monitoring for stuck detection
  > Minimize memory usage

- [ ] POLISH-005: Create README and documentation
  > Goal: Update README.md for Go version
  > Document all CLI commands and flags
  > Add examples for TUI, headless, and CI usage

