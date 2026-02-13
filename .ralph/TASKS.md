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

- [x] AGENT-003: Implement Auggie agent plugin
  > Goal: Create auggie.go implementing Agent interface per AUGGIE_INTEGRATION.md
  > Register with AgentRegistry on init
  > Handle session token authentication (AUGMENT_SESSION_AUTH)
  > Execute with --print --quiet, support --continue for session resumption
  > Parse `auggie models list` for model discovery

- [x] AGENT-004: Add agent discovery and selection
  > Goal: On startup, detect all available agents
  > If multiple available, prompt user to choose (no silent defaults)
  > Store selection in session state

- [x] AGENT-005: Add agent add command
  > Goal: `ralph agent add` for adding custom agents
  > Prompt for: name, command, detection method, model list command
  > Store in config for future sessions

---

## 🔗 Phase 5: Hook System

- [x] HOOK-001: Define hook interface and types
  > Goal: Create hook.go with Hook interface
  > Define HookType (Pre/Post), HookConfig, HookDefinition
  > Support shell and agent hook types
  > Define failure modes: skip_task, warn_continue, abort_loop, ask_agent

- [x] HOOK-002: Implement shell command hooks
  > Goal: Create shell.go for shell command execution
  > Set environment variables (TASK_ID, TASK_NAME, TASK_STATUS, etc.)
  > Capture output and handle errors per on_failure mode

- [x] HOOK-003: Implement agent call hooks
  > Goal: Create agenthook.go for agent-based hooks
  > Run agent with custom prompt
  > Support optional model and agent specification

- [x] HOOK-004: Add hook manager
  > Goal: Create HookManager for hook orchestration
  > Execute pre-task and post-task hooks in order
  > Handle failures according to configured failure modes

---

## 🖥️ Phase 6: TUI Foundation

> **Note**: TUI foundation must be built before Project Analysis TUI components.
> The basic Bubble Tea app structure is needed for confirmation forms and progress displays.

- [x] TUI-001: Create basic Bubble Tea app with header and progress
  > Goal: Create tui/app.go with Model, Init, Update, View
  > Define messages for state updates
  > Basic key handling (q to quit)
  > Create components/header.go - display project name, agent, model, session ID
  > Create components/progress.go - show completed/remaining task counts
  > Style with Lip Gloss
  > Create reusable form components (text input, checkbox, dropdown) for later use

- [x] TUI-002: Implement form components
  > Goal: Create reusable form building blocks for confirmation screens
  > Create components/form.go - form container with field navigation
  > Create components/textinput.go - editable text field
  > Create components/checkbox.go - toggle checkbox
  > Create components/button.go - clickable button with focus state
  > Tab/Shift+Tab navigation between fields
  > Enter to activate/edit, Esc to cancel
  > These will be used by analysis confirmation (BUILD-001a) and task list editor (BUILD-001b)

---

## 🔨 Phase 7: Project Analysis & Build System

- [x] BUILD-000: Refactor existing bootstrap detection to prepare for AI-driven analysis
  > Goal: Remove hardcoded language patterns from internal/build/bootstrap.go
  > This file currently has ~390 lines of pattern-based detection (Go, Node, Python, Rust)
  > Changes needed:
  >   - Remove detectGoProject(), detectNodeProject(), detectPythonProject(), detectRustProject() methods
  >   - Remove DetectProjectType() method and ProjectType enum (will come from AI)
  >   - Remove detectAuto() method that switches on project type
  >   - Keep BootstrapState struct - it will be populated from ProjectAnalysis
  >   - Keep detectManual() for users who want explicit control
  >   - Simplify BootstrapDetector to accept ProjectAnalysis instead of doing detection
  > Update internal/config/config.go:
  >   - Keep BootstrapDetection enum but change semantics:
  >     - "auto" → use AI-driven ProjectAnalysis (new default behavior)
  >     - "manual" → use bootstrap_check command (keep as-is)
  >     - "disabled" → always ready (keep as-is)
  >   - Document that "auto" now means AI-driven, not pattern-based
  > Keep helper methods (fileExists, dirExists) - may be useful elsewhere
  > Tests: Update bootstrap_test.go to reflect simplified interface

- [x] BUILD-001: Implement Project Analysis Agent
  > Goal: Create analysis.go with AI-driven project detection
  > Run an implicit "analysis" agent before the task loop starts
  > Agent prompt asks for structured JSON with project characteristics
  > Parse response into ProjectAnalysis struct (see FEATURES.md Section 4)
  > Detect: project type, languages, build/test commands, greenfield state
  > Provide progress feedback during analysis (TUI spinner with status messages)
  > Cache confirmed analysis in .ralph/project_analysis.json for session
  > Re-run analysis if project structure changes significantly
  > Fallback: If AI analysis fails, use minimal defaults and warn user
  > Reference: FEATURES.md Section 4 "Project Analysis Agent"

- [x] BUILD-001a: Create TUI analysis confirmation form
  > Goal: Present AI analysis results in an editable form for user confirmation
  > Uses TUI-001 app structure and TUI-002 form components from Phase 6
  > After analysis completes, show results in a form-style view:
  >   - Project Type: [Go         ▼]  (dropdown or text field)
  >   - Build Command: [go build ./...]  (editable text)
  >   - Test Command: [go test ./...]   (editable text)
  >   - Build Ready: [✓] Yes  [ ] No
  >   - Test Ready: [✓] Yes  [ ] No
  >   - Greenfield: [ ] Yes  [✓] No
  > Show AI's reasoning/context below each field (collapsed by default)
  > Keyboard navigation: Tab between fields, Enter to confirm, Esc to re-analyze
  > "Confirm & Start" button to proceed with (possibly modified) settings
  > "Re-analyze" button to run analysis again
  > Save user modifications back to ProjectAnalysis before proceeding
  > In headless mode: Skip confirmation, use AI results directly (log them)
  > Reference: FEATURES.md Section 4 "Interactive Confirmation"

- [x] BUILD-001b: Add task list detection and initialization
  > Goal: Detect or create task list as part of initial setup flow
  > **Part 1 - Detection** (in Project Analysis Agent):
  >   - Extend analysis prompt to detect existing task lists in repo
  >   - Look for: TASKS.md, TODO.md, .ralph/tasks.json, GitHub issues, etc.
  >   - Add to ProjectAnalysis: task_list_detected, task_list_path, task_list_format
  > **Part 2 - Auto-import** (if task list found):
  >   - Use agent to parse detected file into our JSON format
  >   - Show parsed tasks in confirmation form for review
  > **Part 3 - Manual initialization** (if no task list found):
  >   - TUI offers options: "Point to file", "Paste list", "Describe goal"
  >   - "Point to file": File picker or path input, agent parses it
  >   - "Paste list": Text area, agent parses pasted content
  >   - "Describe goal": Text area, agent generates task list from description
  > **Part 4 - Task list confirmation form**:
  >   - Show generated/parsed tasks in editable list
  >   - User can add, remove, reorder, edit tasks
  >   - "Confirm" saves to .ralph/tasks.json
  > In headless mode: Require --tasks flag pointing to file, or existing tasks.json
  > Reference: FEATURES.md Section 4 "Task List Initialization"

- [x] BUILD-002: Implement build verification
  > Goal: Create build.go with build execution logic
  > Use ProjectAnalysis.Build.Command (from AI detection)
  > Check ProjectAnalysis.Build.Ready - skip gracefully if not ready
  > Support config override: if build.command is set, use that instead
  > Parse build output for errors
  > Return structured BuildResult with bootstrap awareness

- [x] BUILD-003: Implement test verification
  > Goal: Add test execution with configurable commands
  > Use ProjectAnalysis.Test.Command (from AI detection)
  > Check ProjectAnalysis.Test.Ready - skip gracefully if no test files
  > Support config override: if test.command is set, use that instead
  > Primary: Use exit codes for pass/fail
  > Optional: Custom parsing for detailed test names
  > Return structured TestResult with bootstrap awareness

- [x] BUILD-004: Add TDD mode support
  > Goal: Implement test baseline capture and comparison
  > Use ProjectAnalysis.Test.Ready to determine bootstrap phase
  > Auto-capture baseline when ProjectAnalysis indicates tests exist
  > Global baseline by default (captured once at start)
  > Track newly passing vs regressing tests
  > Store baseline in .ralph/test_baseline.json
  > Block only on regressions, not pre-existing failures
  > Log clear messages: "No tests yet", "Baseline captured", "N regressions detected"
  > Reference: FEATURES.md Section 5 "TDD Mode"

- [x] BUILD-005: Create verification gate logic
  > Goal: Orchestrate project analysis → build → test → gate decision
  > Use ProjectAnalysis for all bootstrap/readiness checks
  > Support gate, tdd, and report modes
  > Handle transitions: bootstrap → ready (log and capture baseline)
  > Parse task metadata for gate overrides (Tests: Not required, Build: Not required)
  > Skip gates when task metadata says not required (log info message)
  > Clear error messaging for failures
  > Return GateResult: { Passed, Skipped (bootstrap), SkippedByTask, Failed, Reason }

---

## 🔄 Phase 8: Main Loop & Session Management

- [x] LOOP-001: Create loop state machine
  > Goal: Define LoopState enum and transitions
  > States: Idle, Running, Paused, AwaitingFix, Completed, Failed
  > Handle state persistence for resume

- [x] LOOP-002: Implement core loop execution
  > Goal: Create loop.go with main execution logic
  > Run Project Analysis Agent FIRST (before any tasks)
  > Inject ProjectAnalysis context into agent prompts
  > Integrate: analysis → task selection → hooks → agent → verify → commit
  > Respect iteration limits per task

- [x] LOOP-003: Add automatic commit logic
  > Goal: Create git.go for git operations
  > Stage changes, create commit with task reference
  > Handle uncommitted changes detection
  > Configurable commit prefix (default: "[ralph]")

- [x] LOOP-004: Implement error recovery
  > Goal: Handle agent failures with retry logic
  > Automatic fix attempts for build/test failures
  > Save state on interruption for resume

- [x] LOOP-005: Implement session management
  > Goal: Generate unique session IDs
  > Persist session state to .ralph/sessions/<id>.json
  > Support `ralph run --continue` for resuming sessions
  > Store agent session IDs for `auggie --continue`

- [x] LOOP-006: Add headless runner
  > Goal: Implement headless execution mode
  > Support --output json for structured output
  > Same functionality as TUI mode without interactive UI
  > Suitable for CI/GitHub Actions

---

## 🖥️ Phase 9: TUI Main Loop Views

> **Note**: These build on Phase 6 TUI Foundation. Phase 6 provides the app shell and form components.
> This phase adds the main loop views: task list, log viewport, status bar, etc.

- [x] TUI-003: Implement task list and status bar components
  > Goal: Create components/taskList.go
  > Scrollable task list with status icons (✓ ○ → ⊘ ⏸ ✗)
  > Highlight current task, support j/k or arrow key navigation
  > Create components/status.go - elapsed time, iteration count, build/test status
  > Display keyboard shortcuts in status bar

- [x] TUI-004: Implement log viewport component
  > Goal: Create components/log.go
  > Real-time agent output streaming
  > Scrollable with auto-follow
  > Option to open in $EDITOR

- [x] TUI-005: Implement task editor and model picker
  > Goal: Create components/taskEditor.go
  > Add new tasks inline (e key), edit task name/description
  > Reorder tasks, save to JSON storage
  > Create components/modelPicker.go
  > List available models from current agent (m key)
  > Show current model indicator

- [x] TUI-006: Add keyboard controls and loop integration
  > Goal: Handle p (pause), s (skip), a (abort), l (logs), h (help)
  > Confirmation dialogs for destructive actions, help overlay
  > Connect loop events to TUI updates
  > Real-time progress updates, error state visualization
  > Pause/resume integration with session management

---

## 🧠 Phase 10: Agent Instructions

- [x] INSTR-001: Create prompt builder ✅
  > Goal: Build agent prompts from template layers
  > Combine base_prompt + platform_prompt + project_prompt + task
  > Inject ProjectAnalysis context (build commands, project type, etc.)
  > Include relevant context from docs and previous changes
  > **Completed:** Created TaskPromptBuilder in internal/prompt/task_builder.go
  > - Combines template layers with analysis + docs + changes context
  > - Refactored loop.go to use TaskPromptBuilder
  > - Comprehensive tests in task_builder_test.go

- [x] INSTR-002: Add plan evolution instructions ✅
  > Goal: Instruct agents to update docs/tasks when plans change
  > "Update remaining tasks if implementation changes the plan"
  > "Document patterns and learnings in project files"
  > **Completed:** Added Phase 6: Plan Evolution to base_prompt.txt (v2.3.0)
  > - 6.1 Update Remaining Tasks when implementation changes approach
  > - 6.2 Document Patterns and Learnings in project docs
  > - 6.3 Keep Future Agents Informed with actionable notes
  > - Added critical rule #10 about updating the plan

---

## 🧪 Phase 11: Comprehensive Testing

> **Note**: Unit tests are written alongside each task (per base_prompt.txt Phase 4).
> This phase adds **comprehensive integration tests** and improves coverage.

- [x] TEST-001: Add unit tests for configuration
  > Goal: Test config loading, validation, defaults
  > Test environment variable overrides
  > Test timeout configuration
  > **Completed:** Added comprehensive tests to config_test.go and loader_test.go:
  > - Custom agent validation (missing name, missing command, invalid detection method)
  > - All valid enum values (DetectionMethod, HookType, FailureMode, BootstrapDetection, TestMode, BaselineScope)
  > - Save function tests (basic, nested directories, with hooks, with custom agents)
  > - Environment variable overrides for all settings (build, test, git)
  > - Edge cases (empty config, partial config, zero timeouts)
  > - Total: 131 test cases in config package

- [ ] TEST-002: Add unit tests for task management
  > Goal: Test JSON storage read/write
  > Test task status updates
  > Test task import from various formats

- [ ] TEST-003: Add unit tests for agents
  > Goal: Mock agent execution
  > Test AgentRegistry plugin system
  > Test model listing, error handling
  > Test session continuation
  > Test Project Analysis Agent response parsing and fallback behavior

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
  > Test analysis confirmation form (field navigation, editing, confirm/re-analyze)
  > Use Bubble Tea testing utilities

---

## 📦 Phase 12: Installation & First-Run Experience

> **Note**: This is the first thing users experience. Make it seamless and delightful.
> Goal: User runs `ralph` in any project → everything "just works"

- [ ] INSTALL-001: Implement zero-config first run
  > Goal: Running `ralph` or `ralph run` in a project without `.ralph/` triggers setup
  > Detect missing `.ralph/` directory
  > Instead of error, show friendly welcome and start setup flow:
  >   1. Create `.ralph/` directory structure
  >   2. Run Project Analysis Agent (BUILD-001)
  >   3. Show analysis confirmation form (BUILD-001a)
  >   4. Detect/create task list (BUILD-001b)
  >   5. Show task list confirmation
  >   6. Start the loop
  > All in one seamless flow - no separate `ralph init` required
  > Save config.yaml with confirmed settings

- [ ] INSTALL-002: Implement explicit `ralph init` command
  > Goal: For users who want to set up without running
  > `ralph init` - interactive setup (same flow as INSTALL-001, but stops before loop)
  > `ralph init --yes` - non-interactive, use AI defaults
  > `ralph init --config config.yaml` - use provided config
  > `ralph init --tasks TASKS.md` - point to existing task file
  > If `.ralph/` exists: prompt to reconfigure or exit
  > `ralph init --force` - overwrite existing config

- [ ] INSTALL-003: Add installation documentation and distribution
  > Goal: Make ralph easy to install globally
  > Support `go install github.com/wexinc/ralph@latest`
  > Create GitHub releases with pre-built binaries (goreleaser)
  > Add install instructions to README:
  >   - macOS: `brew install ralph` (if we publish to homebrew)
  >   - Linux/macOS/Windows: Download binary from releases
  >   - Go users: `go install`
  > Create install script for curl-based install:
  >   `curl -fsSL https://ralph.dev/install.sh | sh`

- [ ] INSTALL-004: Add update and version management
  > Goal: Easy updates and version checking
  > `ralph version` - show current version
  > `ralph update` - check for and install updates (if installed via our script)
  > `ralph run` shows update notification if new version available (non-blocking)
  > Store version info in `.ralph/version.json` for compatibility checks

- [ ] POLISH-001: Add comprehensive error messages
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

