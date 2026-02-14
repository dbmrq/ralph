# Ralph

**AI-Powered Task Automation**

Ralph is an AI-powered task automation tool that runs in a loop, completing tasks from a task list using AI coding agents. It features a beautiful TUI (Terminal User Interface), supports multiple AI agents, and includes intelligent build verification, automatic commits, and graceful error recovery.

## 🚀 Installation

### Homebrew (macOS/Linux)

```bash
brew install wexinc/tap/ralph
```

### Go Install

```bash
go install github.com/wexinc/ralph/cmd/ralph@latest
```

### Binary Download

Download pre-built binaries from the [releases page](https://github.com/wexinc/ralph/releases).

Or use the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/wexinc/ralph/main/scripts/install.sh | bash
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

# Run the automation loop
ralph run                     # Interactive TUI mode
ralph run --headless          # Headless mode (for CI)
ralph run --headless --output json  # JSON output
ralph run --continue          # Resume previous session

# Agent management
ralph agent list              # List available agents
ralph agent add               # Add a custom agent

# Version info
ralph --version
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
| 1. Global | `core/base_prompt.txt` | Ralph workflow instructions |
| 2. Platform | `.ralph/platform_prompt.txt` | Platform-specific guidelines |
| 3. Project | `.ralph/project_prompt.txt` | Your project's unique requirements |

## 🔧 Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/wexinc/ralph.git
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

