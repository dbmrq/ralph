# Ralph Go - Feature Specification

## 1. Agent Plugin System

Agents are implemented as plugins that register with a central registry. This architecture makes adding new agents straightforward without modifying core code.

### Agent Discovery & Selection
- On startup, detect all available agents
- If multiple agents available, **prompt user to choose** (no silent defaults)
- Selected agent persists in session; can be changed via config or TUI

### Cursor Agent Plugin
- **Detection**: Check for `agent` command
- **Models**: Parse `agent --list-models` output
- **Execution**: `agent --print --force [--model MODEL] < prompt`
- **Output**: Real-time streaming to log file and TUI

### Auggie Agent Plugin (Augment CLI)
- **Detection**: Check for `auggie` command
- **Authentication**: Verify via `auggie tokens print` or `AUGMENT_SESSION_AUTH`
- **Models**: `auggie models list` for model discovery
- **Execution**: `auggie --print --quiet [--model MODEL] "prompt"` or pipe via stdin
- **Session Continuation**: `auggie --continue` for resuming paused tasks
- **JSON Output**: `auggie -p --output-format json` for structured parsing
- See `AUGGIE_INTEGRATION.md` for full details

### Adding New Agents
New agents can be added via CLI command:
```bash
ralph agent add
# Prompts for: name, command, detection method, model listing command, etc.
```

Or by implementing the Agent interface in Go and registering with the registry.

## 2. Model Selection

### Model Discovery
- `auggie models list` for Auggie
- `agent --list-models` for Cursor
- Cache model lists (refresh on request)

### Model Configuration
- Per-project default in config
- Runtime selection via TUI model picker or CLI flag
- If not specified: prompt user at startup or use agent's default

## 3. Pre/Post Task Hooks

### Shell Command Hooks
```yaml
hooks:
  pre_task:
    - type: shell
      command: "echo 'Starting task: ${TASK_ID}'"
      on_failure: warn_continue  # skip_task | warn_continue | abort_loop | ask_agent
    - type: shell
      command: "./scripts/prepare.sh"
      on_failure: skip_task
  post_task:
    - type: shell
      command: "npm run lint:fix"
      on_failure: warn_continue
```

### Agent Call Hooks
```yaml
hooks:
  post_task:
    - type: agent
      prompt: "Review the changes made for ${TASK_ID} and suggest improvements"
      model: "claude-sonnet"  # Optional, uses main agent's model if not specified
      agent: "auggie"         # Optional, uses main agent if not specified
      on_failure: ask_agent   # Let the agent decide what to do
```

### Hook Failure Modes
- `skip_task`: Skip the current task and move to next
- `warn_continue`: Log warning but continue with the task
- `abort_loop`: Stop the entire loop
- `ask_agent`: Include failure info in agent prompt, let agent decide

### Hook Environment Variables
- `TASK_ID`: Current task identifier
- `TASK_NAME`: Task name
- `TASK_DESCRIPTION`: Task description
- `TASK_STATUS`: Result status (pending, completed, failed, etc.)
- `ITERATION`: Current iteration number
- `PROJECT_DIR`: Project root directory

## 4. Project Analysis Agent (AI-Driven Detection)

### Problem
Greenfield projects start with no buildable code or tests. Traditional build/test gates would fail immediately because there's nothing to build or test yet. Additionally, hardcoded language-specific patterns are brittle, require maintenance, and can't handle new languages or custom build systems.

### Solution: AI-Driven Project Analysis
Ralph Go uses the AI agent itself to analyze and understand the project. Before starting the task loop, a **Project Analysis Agent** runs with a structured prompt to detect project characteristics dynamically.

This approach has several advantages:
- **Language-agnostic**: Works with any language, including future ones
- **Context-aware**: AI understands project context, not just file patterns
- **Zero configuration**: No user setup required
- **Adaptable**: Handles complex scenarios (monorepos, polyglot projects, custom build systems)
- **Self-improving**: As AI models improve, detection improves automatically

### Project Analysis Agent
Before the first task, Ralph runs an implicit "analysis" phase:

```
┌─────────────────────────────────────────────────────────────────┐
│  RALPH LOOP START                                                │
│                                                                  │
│  1. Project Analysis Agent runs (once per session)               │
│     → Analyzes codebase structure                                │
│     → Returns structured ProjectAnalysis JSON                    │
│                                                                  │
│  2. Loop begins with detected configuration                      │
│     → Build/test commands from analysis                          │
│     → Bootstrap state from analysis                              │
│     → Language-specific context injected into prompts            │
└─────────────────────────────────────────────────────────────────┘
```

### Project Analysis Prompt
The agent receives a prompt like:

```
Analyze this project and return a JSON object with the following structure:

{
  "project_type": "go|node|python|rust|java|mixed|unknown",
  "languages": ["go", "typescript"],  // All languages detected
  "is_greenfield": true,              // No buildable code yet
  "is_monorepo": false,               // Multiple packages/projects

  "build": {
    "ready": false,                   // Can we build?
    "command": "go build ./...",      // Detected or null
    "reason": "No source files yet"   // Human-readable explanation
  },

  "test": {
    "ready": false,                   // Are there tests to run?
    "command": "go test ./...",       // Detected or null
    "has_test_files": false,
    "reason": "No test files found"
  },

  "lint": {
    "command": "golangci-lint run ./...",  // Detected or null
    "available": true
  },

  "dependencies": {
    "manager": "go mod",              // Package manager detected
    "installed": true                 // Dependencies installed?
  },

  "project_context": "This is a Go CLI application using Cobra and Bubble Tea for TUI..."
}

Instructions:
1. Examine the project structure (files, directories, config files)
2. Look for build system markers (go.mod, package.json, Cargo.toml, etc.)
3. Detect what commands would build/test this project
4. Determine if the project is in a "greenfield" state (nothing to build yet)
5. Return ONLY the JSON object, no other text
```

### ProjectAnalysis Go Type
```go
type ProjectAnalysis struct {
    ProjectType   string   `json:"project_type"`
    Languages     []string `json:"languages"`
    IsGreenfield  bool     `json:"is_greenfield"`
    IsMonorepo    bool     `json:"is_monorepo"`

    Build         BuildAnalysis       `json:"build"`
    Test          TestAnalysis        `json:"test"`
    Lint          LintAnalysis        `json:"lint"`
    Dependencies  DependencyAnalysis  `json:"dependencies"`

    ProjectContext string `json:"project_context"`
}

type BuildAnalysis struct {
    Ready   bool    `json:"ready"`
    Command *string `json:"command"`  // nil if not detected
    Reason  string  `json:"reason"`
}

type TestAnalysis struct {
    Ready        bool    `json:"ready"`
    Command      *string `json:"command"`
    HasTestFiles bool    `json:"has_test_files"`
    Reason       string  `json:"reason"`
}
```

### Fallback: Manual Override
Users can still override with explicit configuration if needed:

```yaml
build:
  command: "./custom-build.sh"  # Explicit override, skip AI detection

test:
  command: "./custom-test.sh"   # Explicit override
```

When explicit commands are provided, the analysis agent still runs but those fields are overridden.

### Interactive Confirmation (TUI)

After the AI analysis completes, results are presented in an **editable form** for user confirmation. This ensures transparency and allows users to tweak settings before the task loop begins.

#### Analysis Progress Feedback
During analysis, the TUI shows real-time progress:
```
┌─────────────────────────────────────────────────────────────┐
│  🔍 Analyzing Project...                                    │
│                                                             │
│  ⠼ Running AI analysis...                          00:03   │
│                                                             │
│  ✓ Detected project structure                               │
│  ✓ Found go.mod, package.json                               │
│  ⠼ Determining build commands...                            │
└─────────────────────────────────────────────────────────────┘
```

#### Confirmation Form
Once analysis completes, an editable form is displayed:
```
┌─────────────────────────────────────────────────────────────┐
│  📋 Project Analysis Results                    [Re-analyze]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Project Type    [ Go CLI                    ▼]             │
│  Languages       [ go                          ]            │
│                                                             │
│  Build Command   [ go build ./cmd/ralph        ]            │
│  Build Ready     [✓]                                        │
│                                                             │
│  Test Command    [ go test -race ./...         ]            │
│  Tests Ready     [✓]                                        │
│                                                             │
│  Greenfield      [ ]  (no buildable code yet)               │
│                                                             │
│  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄ │
│  ▶ AI Reasoning (click to expand)                           │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│           [ Confirm & Start ]    [ Re-analyze ]             │
└─────────────────────────────────────────────────────────────┘

Tab: next field | Enter: confirm | Esc: cancel | r: re-analyze
```

#### Form Behavior
- **Tab/Shift+Tab**: Navigate between fields
- **Enter on field**: Edit text fields, toggle checkboxes
- **Enter on "Confirm & Start"**: Accept settings and begin task loop
- **r or "Re-analyze" button**: Run AI analysis again
- **Esc**: Cancel and exit (with confirmation prompt)

#### Headless Mode
In headless mode (`--headless`), the confirmation form is skipped:
- AI analysis results are logged to stdout
- Settings are used directly without user confirmation
- Use `--yes` flag to suppress any prompts

```bash
ralph run --headless
# Output:
# [INFO] Project Analysis:
# [INFO]   Type: Go CLI
# [INFO]   Build: go build ./cmd/ralph (ready)
# [INFO]   Test: go test -race ./... (ready)
# [INFO] Starting task loop...
```

#### Persisted Settings
User modifications are saved to `.ralph/project_analysis.json`. On subsequent runs:
- If cached analysis exists and is recent, show form pre-filled with cached values
- User can still modify and re-analyze if needed
- "Re-analyze" always runs fresh AI analysis

### Task List Initialization

The Project Analysis Agent also detects existing task lists in the repository. If found, they are automatically parsed; if not, the user is offered several ways to create one.

#### Auto-Detection
The analysis agent looks for existing task lists:
- `.ralph/tasks.json` (our native format)
- `TASKS.md`, `TODO.md`, `ROADMAP.md` (markdown task lists)
- `.github/ISSUES.md` or linked GitHub issues
- `docs/tasks.md`, `docs/TODO.md`

If found, the `ProjectAnalysis` includes:
```json
{
  "task_list": {
    "detected": true,
    "path": "TASKS.md",
    "format": "markdown",
    "task_count": 15
  }
}
```

#### Auto-Import Flow
When a task list is detected:
1. Agent parses the file into our JSON format
2. Parsed tasks shown in confirmation form for review
3. User can edit, reorder, add, or remove tasks
4. "Confirm" saves to `.ralph/tasks.json`

#### Manual Initialization (No Task List Found)
If no task list is detected, the TUI offers options:
```
┌─────────────────────────────────────────────────────────────┐
│  📋 No Task List Found                                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  How would you like to create your task list?               │
│                                                             │
│  [ 1 ] Point to a file                                      │
│        Browse or enter path to existing task file           │
│                                                             │
│  [ 2 ] Paste a list                                         │
│        Paste tasks from clipboard or type them              │
│                                                             │
│  [ 3 ] Describe your goal                                   │
│        Describe what you want to build, AI generates tasks  │
│                                                             │
│  [ 4 ] Start empty                                          │
│        Begin with no tasks, add them manually               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Option 1: Point to a file**
- File picker or path input
- Agent parses any format (markdown, plain text, JSON, YAML)
- Shows parsed tasks for confirmation

**Option 2: Paste a list**
- Text area for pasting
- Agent parses pasted content (any format)
- Shows parsed tasks for confirmation

**Option 3: Describe your goal**
- Text area for natural language description
- Example: "Build a REST API with user authentication, database integration, and admin dashboard"
- Agent generates a structured task list
- Shows generated tasks for confirmation and editing

**Option 4: Start empty**
- Creates empty `.ralph/tasks.json`
- User can add tasks via TUI task editor

#### Task List Confirmation Form
After parsing/generating, tasks are shown in an editable list:
```
┌─────────────────────────────────────────────────────────────┐
│  📋 Task List (12 tasks)                        [Re-parse]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ○ TASK-001: Set up project structure                       │
│    > Initialize Go module, create directories               │
│                                                             │
│  ○ TASK-002: Add core dependencies                          │
│    > Add Cobra, Viper, Bubble Tea                           │
│                                                             │
│  ○ TASK-003: Create CLI skeleton                            │
│    > Implement root command with subcommands                │
│                                                             │
│  ... (scroll for more)                                      │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  [a]dd  [e]dit  [d]elete  [↑↓]move  [Enter]confirm          │
└─────────────────────────────────────────────────────────────┘
```

#### Headless Mode
In headless mode, task list must be provided:
```bash
# Use existing tasks.json
ralph run --headless

# Point to a file (agent parses it)
ralph run --headless --tasks ./TASKS.md

# Pipe tasks via stdin
cat tasks.txt | ralph run --headless --tasks -
```

### Context Injection into Task Prompts

The `ProjectAnalysis.ProjectContext` field provides rich context that is injected into every task agent prompt. This enables:

- **Language-aware instructions**: Agent knows "this is a Go project using Cobra for CLI"
- **Build/test commands**: Agent knows exactly how to verify their work
- **Dependency context**: Agent knows what tools are available

Example injection into prompt:
```
# Project Context (from analysis)

Project Type: Go CLI application
Languages: Go
Build Command: go build ./cmd/ralph
Test Command: go test -race -cover ./...
Package Manager: go mod
Dependencies Installed: Yes

This is a Go CLI application using Cobra for command handling and Bubble Tea
for TUI. The project follows standard Go project layout with cmd/ and internal/
directories.

---

# Your Task
...
```

### Re-Analysis Triggers

The Project Analysis Agent runs once per session by default, but re-runs when:
- User explicitly requests it (TUI command or CLI flag)
- Session detects major file structure changes (new package.json, go.mod, etc.)
- Previous analysis is older than configurable threshold (default: 24 hours)

### Bootstrap State Behavior
- **Build gate**: Skip with info message, exit 0
- **Test gate**: Skip with info message, exit 0
- **TDD mode**: No baseline captured until tests exist
- **Logging**: Clear indication that bootstrap phase is active

### Task-Level Gate Overrides
Individual tasks can specify that tests or builds are not required using metadata in the task description:

```markdown
- [ ] INIT-001: Initialize project structure
  > Goal: Create go.mod and basic files
  > Tests: Not required (setup-only task)
  > Build: Not required

- [ ] INIT-002: Add dependencies
  > Tests: Not required (dependency-only task)
```

Supported patterns (case-insensitive):
- `Tests: Not required` / `Tests: None` / `Tests: N/A` / `Tests: Skip`
- `Build: Not required` / `Build: None` / `Build: N/A` / `Build: Skip`
- `No tests needed` / `No tests required`

When a task has these markers, the corresponding gate is skipped with an info message after that task completes.

### Interactive Skip During Initial Checks
During initial build and test verification (before the task loop starts), the user can press **'s'** to skip the check. This is useful when:
- You know the project state is good and don't want to wait
- The check is taking a long time (e.g., Go not in PATH causing fallback behavior)
- You're in a greenfield/bootstrap phase with nothing to verify

The skip option is shown in the spinner:
```
⠼ Running tests... 00:05  (press 's' to skip)
```

When skipped, the loop continues normally without attempting to fix anything.

### Transition Detection
When bootstrap phase ends (first buildable code appears):
1. Run initial build verification
2. If TDD mode: Capture initial test baseline
3. Log transition: "Bootstrap complete, verification gates now active"

## 5. TDD Mode

### Problem
With TDD, tests are written before implementation. Traditional test gates fail because tests don't pass initially.

### Solution
```yaml
test:
  mode: tdd                # gate | tdd | report
  baseline_file: .ralph/test_baseline.json
  baseline_scope: global   # global | session | task
```

### TDD Workflow
1. **Baseline Capture**: Record which tests pass/fail at start (global by default)
2. **Progress Tracking**: Track newly passing tests as progress
3. **Regression Detection**: Fail if previously passing tests now fail
4. **No "all must pass" requirement**: Unlike gate mode, TDD allows failing tests

### Interaction with Bootstrap Detection
- If no test files exist yet: Skip baseline capture, log info message
- When first test file appears: Capture initial baseline automatically
- Baseline is updated only when explicitly requested or on session start

### Test Result Capture
- **Primary method**: Exit codes (works for Go, npm, pytest, etc.)
- **Optional**: Custom parsing for detailed test names (configurable)

### Test Baseline Format
```json
{
  "captured_at": "2026-02-13T10:00:00Z",
  "scope": "global",
  "passing": ["TestAuth", "TestUser"],
  "failing": ["TestNewFeature", "TestEdgeCase"],
  "skipped": [],
  "bootstrap_completed_at": "2026-02-13T09:30:00Z"
}
```

### TDD Gate Logic
```
if no_tests_exist:
    return SKIP (bootstrap phase)

if no_baseline_exists:
    capture_baseline()
    return PASS (first run, nothing to regress)

current_results = run_tests()
regressions = baseline.passing - current_results.passing

if regressions:
    return FAIL (regression detected: {regressions})
else:
    return PASS (no regressions, {newly_passing} tests now passing)
```

## 6. Self-Improving Feedback Loop

### Simplified Approach
Instead of complex feedback storage, agents are instructed to **update existing files directly**:
- Update documentation with learnings
- Add context to task descriptions
- Document patterns in project files

This keeps learnings in context and avoids stale feedback accumulation.

### Plan Evolution
When implementation changes the plan:
- Agents update remaining tasks in the task list
- Agents update project documentation
- Changes are committed with the task

## 7. Session Management (Pause/Resume)

### Pausing Tasks
- User presses `p` in TUI or sends SIGINT
- Current task state saved with session ID
- For Auggie: session ID stored for `--continue` flag

### Resuming Tasks
- `ralph run --continue` or select paused session in TUI
- Agents resume from saved state
- Context includes what was completed before pause

## 8. Error Handling & Edge Cases

### Smart Timeout System
```yaml
timeout:
  active: 2h    # While agent is producing output
  stuck: 30m    # No output change threshold
```

System monitors output to detect stuck vs active agents.

### Agent Failures
- Retry logic with exponential backoff
- Save state on failure for resume
- Clear error messaging with suggestions

### Build/Test Failures
- Detailed error extraction from logs
- Automatic fix attempts (configurable count)
- Human escalation option via TUI prompt

### Git Conflicts
- Detect before starting task
- TUI prompt to resolve or abort

### State Recovery
- Save loop state to disk after each iteration
- Resume from last successful state
- Handle partial task completions

## 9. Git Integration

### Auto-Commit Behavior
```yaml
git:
  auto_commit: true           # After each successful task (default)
  commit_prefix: "[ralph]"
  push: false                 # Manual push by default
```

### Commit Messages
Auto-generated with task reference:
```
[ralph] Complete TASK-005: Implement error handling
```

## 10. TUI Components

### Main View
```
╔══════════════════════════════════════════════════════════════════╗
║  RALPH LOOP  │  Project: my-app  │  Agent: auggie  │  opus-4    ║
╠══════════════════════════════════════════════════════════════════╣
║  Progress: ████████░░░░░░░░░░░░  │  5/12 tasks  │  Iteration 6  ║
╠══════════════════════════════════════════════════════════════════╣
║  ✓ TASK-001: Setup project structure                             ║
║  ✓ TASK-002: Add user authentication                             ║
║  ✓ TASK-003: Create API endpoints                                ║
║  ✓ TASK-004: Add validation layer                                ║
║  → TASK-005: Implement error handling  (in progress)             ║
║  ○ TASK-006: Add logging middleware                              ║
║  ○ TASK-007: Write integration tests                             ║
╠══════════════════════════════════════════════════════════════════╣
║  Agent Output:                                                    ║
║  ┌────────────────────────────────────────────────────────────┐  ║
║  │ Analyzing task requirements...                              │  ║
║  │ Reading existing error patterns in codebase...              │  ║
║  │ Creating pkg/errors/errors.go...                            │  ║
║  │ Adding error middleware to router...                        │  ║
║  └────────────────────────────────────────────────────────────┘  ║
╠══════════════════════════════════════════════════════════════════╣
║  [p]ause  [s]kip  [a]bort  [l]ogs  [h]elp   │  Elapsed: 02:45:30 ║
╚══════════════════════════════════════════════════════════════════╝
```

### Task List Features
- Scrollable with j/k or arrow keys
- Add new tasks inline
- Edit/reorder tasks
- Status icons: ✓ completed, → in progress, ○ pending, ⊘ skipped, ⏸ paused, ✗ failed

### Log Viewer
- Scrollable overlay (press `l`)
- Shows log file path
- Option to open in `$EDITOR`

### Model Picker
- List available models from agent
- Select before or during run

### Keyboard Controls
- `p`: Pause/resume after current task
- `s`: Skip current task (mark as skipped, available in future runs)
- `a`: Abort loop (graceful shutdown)
- `l`: Open log viewer
- `h`: Show help
- `q`: Quit (with confirmation if running)
- `r`: Review mode (pause and review changes)
- `e`: Open task editor
- `m`: Model picker

## 11. Headless Mode

Equally supported with TUI for CI/GitHub Actions:

```bash
ralph run --headless
ralph run --headless --output json  # Structured JSON output
ralph run --continue                # Resume previous session
```

### Use Cases
- GitHub Actions/Workflows
- CI/CD pipelines
- Background execution
- Remote automation

