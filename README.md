# Ralph

**AI-Powered Task Automation**

Ralph is an AI-powered task automation tool that runs in a loop, completing tasks from a task list using AI coding agents. It features a beautiful TUI (Terminal User Interface), supports multiple AI agents, and includes intelligent build verification, automatic commits, and graceful error recovery.

## 🚀 Installation

### Homebrew (macOS/Linux)

```bash
brew install dbmrq/tap/ralph
```

### Go Install

```bash
go install github.com/dbmrq/ralph/cmd/ralph@latest
```

### Binary Download

Download pre-built binaries from the [releases page](https://github.com/dbmrq/ralph/releases).

Or use the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/dbmrq/ralph/main/scripts/install.sh | bash
```

## 🎯 Quick Start

```bash
# Navigate to your project
cd your-project

# Initialize Ralph (runs setup wizard)
ralph init

# Or just run - Ralph will initialize automatically on first run
ralph run
```

That's it! Ralph will:
1. Analyze your project using AI
2. Detect build and test commands
3. Ask you to confirm or edit the detected settings
4. Help you set up a task list
5. Start automating!

## ✨ Features

- 🖥️ **Beautiful TUI** - Real-time progress display with Bubble Tea
- 🤖 **Multiple AI Agents** - Supports Cursor, Augment (Auggie), and custom agents
- 🔄 **Automated Task Loop** - Runs until all tasks complete or limits reached
- 🔨 **Smart Build Gates** - AI-detected build/test verification between tasks
- 📝 **Automatic Commits** - Commits each completed task separately
- 🧪 **TDD Support** - Captures test baselines, blocks only on regressions
- 🪝 **Hook System** - Pre/post task hooks (shell commands or agent calls)
- 🛡️ **Safety Limits** - Max iterations, timeout detection, error recovery
- 🌿 **Branch Protection** - Configurable branch restrictions
- 📋 **3-Level Prompts** - Global, platform, and project-specific instructions
- ⏸️ **Session Management** - Pause and resume task automation
- 📊 **Headless Mode** - CI/GitHub Actions support with JSON output

## 📖 Usage

### Commands

```bash
# Initialize Ralph in a project
ralph init                    # Interactive setup
ralph init --yes              # Non-interactive, use AI defaults
ralph init --tasks TASKS.md   # Import existing task file
ralph init --config my.yaml   # Use provided config file
ralph init --force            # Reinitialize, overwriting existing

# Run the automation loop
ralph run                     # Interactive TUI mode
ralph run --headless          # Headless mode (for CI)
ralph run --headless --output json  # JSON output
ralph run --headless --tasks TASKS.md  # Use specific task file
ralph run --continue <id>     # Resume previous session
ralph run --verbose           # Enable verbose logging

# Agent management
ralph agent list              # List available agents
ralph agent add               # Add a custom agent (interactive)
ralph agent add --name myagent --command myagent  # Non-interactive

# Version and updates
ralph version                 # Show detailed version info
ralph version --check         # Check for updates
ralph update                  # Update to latest version
ralph update --check          # Check only, don't install

# Shell completion
ralph completion bash         # Generate bash completions
ralph completion zsh          # Generate zsh completions
ralph completion fish         # Generate fish completions
```

### Keyboard Shortcuts (TUI Mode)

| Key | Action |
|-----|--------|
| `p` | Pause/Resume loop |
| `s` | Skip current task |
| `a` | Abort loop |
| `l` | Toggle log overlay |
| `m` | Model picker |
| `e` | Edit/add task |
| `h` | Help overlay |
| `q` | Quit |

### Command Reference

#### `ralph init`

Initialize Ralph in the current project.

| Flag | Short | Description |
|------|-------|-------------|
| `--yes` | `-y` | Non-interactive mode, use AI defaults |
| `--config` | `-c` | Path to config file to use |
| `--tasks` | `-t` | Path to task file to import |
| `--force` | `-f` | Overwrite existing configuration |

#### `ralph run`

Start the automation loop.

| Flag | Description |
|------|-------------|
| `--headless` | Run without TUI (for CI) |
| `--output` | Output format: `json` (requires `--headless`) |
| `--tasks` | Path to task file (headless mode) |
| `--continue` | Resume session by ID |
| `--verbose` | Enable verbose logging |

#### `ralph agent add`

Add a custom agent.

| Flag | Short | Description |
|------|-------|-------------|
| `--name` | `-n` | Agent name |
| `--command` | `-c` | Agent command |
| `--description` | `-d` | Agent description |
| `--detection` | | Detection method: `command`, `path`, `env`, `always` |
| `--detection-value` | | Value for detection |
| `--model-list-cmd` | | Command to list models |
| `--default-model` | | Default model |

#### `ralph version`

Show version information.

| Flag | Short | Description |
|------|-------|-------------|
| `--check` | `-c` | Check for available updates |

#### `ralph update`

Update to the latest version.

| Flag | Short | Description |
|------|-------|-------------|
| `--check` | `-c` | Only check, don't install |
| `--yes` | `-y` | Don't prompt for confirmation |

## ⚙️ Configuration

Configuration is stored in `.ralph/config.yaml`:

```yaml
agent:
  default: ""  # Empty = prompt if multiple available

timeout:
  active: 2h   # Max time when agent is producing output
  stuck: 30m   # Max time without output

git:
  auto_commit: true
  commit_prefix: "[ralph]"

build:
  command: ""  # Auto-detected by AI if empty

test:
  command: ""  # Auto-detected by AI if empty
  mode: gate   # gate | tdd | report

hooks:
  pre_task: []
  post_task: []
```

## 🤖 Supported AI Agents

### Cursor Agent
- Requires [Cursor IDE](https://cursor.sh) with CLI enabled
- Uses `agent` command

### Augment Agent (Auggie)
- Requires [Augment CLI](https://augmentcode.com)
- Uses `auggie` command
- Supports session continuation with `--continue`

### Custom Agents
Add your own agents with `ralph agent add`:
- Name and description
- Detection command
- Model listing command
- Execution command template

## 📋 Task Format

Tasks can be imported from markdown:

```markdown
- [ ] TASK-001: Implement user authentication
  > Goal: Add login/logout functionality
  > Reference: See docs/auth-spec.md

- [ ] TASK-002: Add form validation
  > Goal: Validate email and password fields
```

Or created interactively through the TUI.

## 🪝 Hooks

Configure pre/post task hooks in `.ralph/config.yaml`:

```yaml
hooks:
  pre_task:
    - type: shell
      command: "echo 'Starting task: ${TASK_ID}'"
      on_failure: warn_continue

  post_task:
    - type: shell
      command: "npm run lint:fix"
      on_failure: warn_continue
    - type: agent
      prompt: "Review the changes and suggest improvements"
      on_failure: ask_agent
```

### Hook Failure Modes
- `skip_task`: Skip the current task and move to next
- `warn_continue`: Log warning but continue
- `abort_loop`: Stop the entire loop
- `ask_agent`: Let the agent decide what to do

## 🧪 TDD Mode

Ralph supports Test-Driven Development workflows:

```yaml
test:
  mode: tdd           # gate | tdd | report
  baseline_scope: global  # global | session | task
```

In TDD mode, Ralph:
1. Captures a baseline of existing test failures
2. Blocks only on test **regressions** (newly failing tests)
3. Allows pre-existing failures to continue

## 🔄 CI/GitHub Actions

Ralph supports headless mode for CI/CD pipelines. Use `--headless` to disable the TUI.

### Basic GitHub Actions Workflow

```yaml
name: Ralph Automation

on:
  workflow_dispatch:  # Manual trigger
    inputs:
      task_limit:
        description: 'Maximum tasks to complete'
        default: '5'

jobs:
  ralph:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install Ralph
        run: go install github.com/dbmrq/ralph/cmd/ralph@latest

      - name: Run Ralph
        env:
          AUGMENT_SESSION_AUTH: ${{ secrets.AUGMENT_SESSION_AUTH }}
        run: |
          ralph run --headless --output json 2>&1 | tee ralph-output.json

      - name: Upload Results
        uses: actions/upload-artifact@v4
        with:
          name: ralph-output
          path: ralph-output.json
```

### JSON Output Format

When using `--output json`, Ralph outputs structured JSON:

```json
{
  "session_id": "abc123",
  "status": "completed",
  "tasks_completed": 3,
  "tasks_remaining": 2,
  "tasks": [
    {
      "id": "TASK-001",
      "name": "Implement feature",
      "status": "completed",
      "iterations": 1
    }
  ],
  "errors": []
}
```

### Headless Mode Environment Variables

| Variable | Description |
|----------|-------------|
| `AUGMENT_SESSION_AUTH` | Augment CLI authentication token |
| `RALPH_TIMEOUT_ACTIVE` | Override active timeout (e.g., "2h") |
| `RALPH_TIMEOUT_STUCK` | Override stuck timeout (e.g., "30m") |
| `RALPH_GIT_AUTO_COMMIT` | Enable/disable auto-commit ("true"/"false") |

## 📂 Project Structure

After running `ralph init`, your project will have:

```
your-project/
├── .ralph/
│   ├── config.yaml           # Configuration
│   ├── tasks.json            # Task storage
│   ├── project_analysis.json # Cached AI analysis
│   ├── sessions/             # Session state
│   ├── logs/                 # Run logs
│   └── docs/                 # Additional documentation
└── (your project files)
```

## 📋 3-Level Prompt System

Instructions are layered for flexibility:

| Level | File | Purpose |
|-------|------|---------|
| 1. Global | `.ralph/base_prompt.txt` | Ralph workflow instructions |
| 2. Platform | `.ralph/platform_prompt.txt` | Platform-specific guidelines |
| 3. Project | `.ralph/project_prompt.txt` | Your project's unique requirements |

## 🔧 Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/dbmrq/ralph.git
cd ralph

# Build
make build

# Run tests
make test

# Install locally
make install
```

### Running Tests

```bash
make test           # Run all tests
make test-verbose   # Verbose output
make test-coverage  # Generate coverage report
```

### Creating a Release

Releases are created using [GoReleaser](https://goreleaser.com/):

```bash
# Create a snapshot (for testing)
make snapshot

# Create a release (requires GITHUB_TOKEN)
make release
```

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests (`make test`)
5. Submit a pull request

## 📄 License

MIT

