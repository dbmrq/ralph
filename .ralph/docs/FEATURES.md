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

## 4. Bootstrap/Greenfield Detection

### Problem
Greenfield projects start with no buildable code or tests. Traditional build/test gates would fail immediately because there's nothing to build or test yet.

### Solution
Ralph Go automatically detects bootstrap state and gracefully skips verification:

```yaml
build:
  bootstrap_detection: auto  # auto | manual | disabled
  # auto: Detect based on project type (no go.mod, no package.json, etc.)
  # manual: Use bootstrap_check command
  # disabled: Always run build/test commands
  bootstrap_check: ""        # Custom command (exit 0 = bootstrap, non-zero = ready)
```

### Bootstrap Detection Logic
1. **Auto-detection** (default): Check for project markers
   - Go: `go.mod` exists AND `*.go` files exist
   - Node: `package.json` exists AND `node_modules/` exists
   - Python: `setup.py` or `pyproject.toml` exists
   - Generic: At least one source file matching configured patterns

2. **Custom check**: Run user-provided command
   - Exit 0 = still in bootstrap phase (skip verification)
   - Non-zero = project ready for verification

3. **Test file detection**: Separate check for test readiness
   - Go: `*_test.go` files exist
   - Node: `*.test.js` or `*.spec.js` files exist
   - Python: `test_*.py` or `*_test.py` files exist

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

