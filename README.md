# Ralph Loop

**Automated AI Agent Task Runner**

Ralph Loop is a shell script that repeatedly calls an AI coding agent to complete tasks from a checklist. It handles iteration limits, build verification, automatic commits, and graceful error recovery.

## 🚀 One-Liner Install

Copy and paste this into your terminal to get started immediately:

```bash
bash <(gh api repos/W508153_wexinc/ralph-loop/contents/install.sh --jq '.content' | base64 -d)
```

### What it does

1. **Downloads** the installer script from the private GitHub repo (using `gh` CLI for authentication)
2. **Asks** where to install ralph-loop (suggests a sensible default based on your current directory)
3. **Clones** the ralph-loop repository to that location
4. **Launches** the interactive setup wizard which:
   - Asks for your project path
   - Detects project type (iOS, React, Python, Node.js, etc.)
   - Auto-detects Xcode schemes for iOS projects
   - Configures build and test commands
   - Lets you choose an AI agent (Cursor, Augment, or custom)
   - Creates a feature branch in your project
   - Generates all `.ralph/` configuration files
5. **Outputs** the exact command to run Ralph Loop on your project

### Requirements

- **GitHub CLI (`gh`)** - Required because this is a private repository
  ```bash
  # Install with Homebrew
  brew install gh

  # Authenticate
  gh auth login
  ```

### If the repo becomes public

If this repo is made public, you can use the simpler curl command:
```bash
curl -fsSL https://raw.githubusercontent.com/W508153_wexinc/ralph-loop/main/install.sh | bash
```

## Features

- 🔄 **Automated task loop** - Runs until all tasks complete or limits reached
- 🔨 **Build gates** - Verifies builds pass between tasks; auto-fixes if broken
- 🤖 **Pluggable agents** - Supports Cursor, Augment (auggie), or custom agents
- 📝 **Automatic commits** - Commits each completed task separately
- 🛡️ **Safety limits** - Max iterations, consecutive failure detection
- 📊 **Detailed logging** - Per-run and per-iteration logs
- 🌿 **Branch protection** - Prevents running on main/master

## Quick Start

### Option A: One-Liner Install (Recommended)

```bash
bash <(gh api repos/W508153_wexinc/ralph-loop/contents/install.sh --jq '.content' | base64 -d)
```

### Option B: Interactive Setup Wizard

If you prefer to clone manually first:

```bash
# Clone Ralph Loop
cd ~/projects
git clone https://github.com/W508153_wexinc/ralph-loop.git

# Run the setup wizard
./ralph-loop/setup.sh
```

The wizard will:
- Ask for your project location
- Detect your project type (iOS, React, Python, etc.)
- Configure build commands automatically
- Create a feature branch
- Generate all configuration files
- Provide exact commands to run

### Option C: Manual Setup

#### 1. Clone Ralph Loop

```bash
cd ~/projects
git clone https://github.com/W508153_wexinc/ralph-loop.git
```

#### 2. Set Up Your Project

```bash
# In your project directory
cd my-ios-app
mkdir -p .ralph

# Copy template files
cp ../ralph-loop/templates/ios/config.sh .ralph/
cp ../ralph-loop/templates/ios/prompt.txt .ralph/
cp ../ralph-loop/templates/ios/TASKS.md .ralph/
```

#### 3. Customize Configuration

Edit `.ralph/config.sh`:
```bash
PROJECT_NAME="My iOS App"
XCODE_SCHEME="MyApp"
COMMIT_SCOPE="ios"
```

Edit `.ralph/prompt.txt` with your project-specific instructions.

Edit `.ralph/TASKS.md` with your task list.

#### 4. Create a Feature Branch

```bash
git checkout -b feature/ralph-tasks
```

#### 5. Run Ralph Loop

```bash
# From parent directory containing both repos
cd ~/projects
./ralph-loop/ralph_loop.sh ./my-ios-app

# Or specify an agent
./ralph-loop/ralph_loop.sh ./my-ios-app auggie
```

## Directory Structure

```
~/projects/
├── ralph-loop/              # This repo
│   ├── ralph_loop.sh        # Main script
│   ├── base_prompt.txt      # General agent instructions
│   └── templates/           # Project templates
│       └── ios/
│           ├── config.sh
│           ├── prompt.txt
│           └── TASKS.md
│
└── my-project/              # Your project
    ├── .ralph/              # Ralph configuration
    │   ├── config.sh        # Project settings
    │   ├── prompt.txt       # Project-specific prompt
    │   ├── TASKS.md         # Task checklist
    │   └── logs/            # Run logs (auto-created)
    └── (your project files)
```

## Configuration Reference

### config.sh Options

| Variable | Default | Description |
|----------|---------|-------------|
| `PROJECT_NAME` | - | Display name for your project |
| `AGENT_TYPE` | `cursor` | Agent to use: `cursor`, `auggie`, `custom` |
| `MAX_ITERATIONS` | `50` | Maximum loop iterations |
| `PAUSE_SECONDS` | `5` | Pause between iterations |
| `MAX_CONSECUTIVE_FAILURES` | `3` | Stop after N consecutive failures |
| `REQUIRE_BRANCH` | `true` | Require non-main branch |
| `ALLOWED_BRANCHES` | `""` | Specific allowed branches (empty = any) |
| `AUTO_COMMIT` | `true` | Auto-commit after each task |
| `COMMIT_PREFIX` | `feat` | Commit message prefix |
| `COMMIT_SCOPE` | `""` | Commit scope, e.g., `ios` |
| `BUILD_GATE_ENABLED` | `true` | Verify builds between tasks |
| `BUILD_FIX_ATTEMPTS` | `1` | Attempts to fix broken builds |

### Build Commands

Define these functions in `config.sh`:

```bash
# Required if BUILD_GATE_ENABLED=true
project_build() {
    xcodebuild -scheme "MyApp" build
}

# Optional
project_test() {
    xcodebuild -scheme "MyApp" test
}
```

### Custom Agents

To use a custom agent, set `AGENT_TYPE="custom"` and define:

```bash
run_agent_custom() {
    local prompt="$1"
    local log_file="$2"

    my-custom-agent --prompt "$prompt" > "$log_file" 2>&1
    cat "$log_file"
}
```

## Task File Format

Tasks use markdown checkbox format:

```markdown
- [ ] TASK-001: Uncompleted task
  > Goal: What this task should accomplish
  > Reference: Link to specs or examples

- [x] TASK-002: Completed task
  > Goal: Already done
```

### Task Writing Tips

1. **One atomic change per task** - Completable in one agent run
2. **Clear success criteria** - Agent knows when it's done
3. **Include references** - Links to designs, specs, examples
4. **Order matters** - Dependencies come first
5. **Consistent IDs** - Format: `PREFIX-###` (e.g., `AUTH-001`)

## Status Markers

Agents must output one of these at the end of their response:

| Marker | Meaning |
|--------|---------|
| `NEXT` | Task completed, more tasks remain |
| `DONE` | Last task completed, all done |
| `ERROR: msg` | Unrecoverable error occurred |
| `FIXED` | Build fix completed (special mode) |

## Build Gate Behavior

When `BUILD_GATE_ENABLED=true`:

1. **Before starting**: Verifies initial build passes
2. **After each task**: Verifies build still passes
3. **On build failure**: Calls agent with special "fix build" prompt
4. **If fix fails**: Loop stops with error

This ensures the project is never left in a broken state.

## Logs

Logs are stored in `.ralph/logs/`:

- `ralph_run_YYYYMMDD_HHMMSS.log` - Master log for the run
- `iteration_YYYYMMDD_HHMMSS_NNN.log` - Individual iteration logs
- `build_fix_YYYYMMDD_HHMMSS.log` - Build fix attempt logs

## Examples

### iOS Project

See `templates/ios/` for a complete iOS setup.

### Running with Different Agents

```bash
# Use Cursor (default)
./ralph_loop.sh ../my-app

# Use Augment
./ralph_loop.sh ../my-app auggie

# Use custom agent (defined in config.sh)
./ralph_loop.sh ../my-app custom
```

### Resuming After Interruption

Just run the script again - it picks up from the first unchecked task:

```bash
./ralph_loop.sh ../my-app
```

## Troubleshooting

### "Cannot run on 'main' branch"

Create a feature branch first:
```bash
cd my-project
git checkout -b feature/my-feature
```

### "Build failed and could not be fixed"

1. Check the build fix log in `.ralph/logs/`
2. Manually fix the build
3. Commit the fix
4. Run Ralph Loop again

### Agent not found

Install the required CLI:
- **Cursor**: Install Cursor IDE, enable CLI
- **Augment**: `npm install -g @anthropic/augment-cli`

## License

MIT

