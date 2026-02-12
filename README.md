# Ralph Loop

**Automated AI Agent Task Runner**

Ralph Loop is a shell script that repeatedly calls an AI coding agent to complete tasks from a checklist. It handles iteration limits, build verification, automatic commits, and graceful error recovery.

## 🚀 Get Started (One Command)

Copy and paste this into your terminal:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/dbmrq/ralph-loop/main/install.sh)
```

**That's it!** This single command handles everything from start to finish.

> **Alternative (if using GitHub CLI):**
> ```bash
> bash <(gh api repos/dbmrq/ralph-loop/contents/install.sh --jq '.content' | base64 -d)
> ```

### What Happens

The script guides you through 100% of the setup:

1. **Checks prerequisites** - Installs Homebrew and GitHub CLI if needed
2. **Authenticates** - Logs you into GitHub if not already authenticated
3. **Configures your project** - Detects project type, sets up build commands
4. **Installs files** - Copies Ralph Loop files into your project's `.ralph/` directory
5. **AI Setup Assistant** - Calls an AI agent to analyze your project and configure settings
6. **Creates a branch** - Sets up a feature branch for safety
7. **Runs Ralph Loop** - Starts the automation when you're ready

### First Time? No Problem!

The installer handles everything, even if you have nothing installed:

```
┌─────────────────────────────────────────────────┐
│  Don't have Homebrew?  → Offers to install it   │
│  Don't have gh CLI?    → Installs via Homebrew  │
│  Not authenticated?    → Walks you through it   │
│  Don't have ralph-loop? → Clones it for you     │
└─────────────────────────────────────────────────┘
```

### Already Have ralph-loop?

Just run the installer again from anywhere:

```bash
# From within the ralph-loop directory
./install.sh

# Or use the one-liner from anywhere
bash <(curl -fsSL https://raw.githubusercontent.com/dbmrq/ralph-loop/main/install.sh)
```

It detects the existing installation and shows you a menu:

```
What would you like to do?
  1) Set up a new project
  2) Add/edit tasks for an existing project
  3) Add custom instructions for an existing project
  4) Run Ralph Loop on a project
  5) Update ralph-loop to latest version
  6) Exit
```

## Features

- 🔄 **Automated task loop** - Runs until all tasks complete or limits reached
- 🔨 **Build gates** - Verifies builds pass between tasks; auto-fixes if broken
- 🤖 **Pluggable agents** - Supports Cursor, Augment (auggie), or custom agents
- 📝 **Automatic commits** - Commits each completed task separately
- 🛡️ **Safety limits** - Max iterations, consecutive failure detection
- ✅ **Test run mode** - Pauses after first 2 tasks for verification before continuing
- 📊 **Detailed logging** - Per-run and per-iteration logs
- 🌿 **Branch protection** - Prevents running on main/master
- 🧙 **Smart installer** - Handles all prerequisites automatically
- 📋 **3-level prompts** - Separate global, platform, and project instructions

## How It Works

### The Single Entry Point

`install.sh` is designed to be THE entry point for Ralph Loop. You never need to remember multiple commands - just run the installer and it figures out what to do:

| Situation | What the installer does |
|-----------|------------------------|
| Fresh install | Installs prerequisites, clones repo, sets up project |
| Already installed | Shows menu of actions |
| Project not configured | Runs the setup wizard |
| Project already configured | Offers to reconfigure or run |
| On main/master branch | Offers to create a feature branch |

### Adding Tasks

When setting up a project, you can add tasks interactively:

```
Enter your tasks one by one.
Format: Brief description of what the agent should do
Type 'done' when finished.

Task 1: Implement the login button action
  Details (optional): Connect to AuthService.login()
✓ Added TASK-001

Task 2: Add form validation for email field
  Details (optional): Use regex validation, show inline errors
✓ Added TASK-002

Task 3: done
```

### Adding Custom Instructions

You can also add project-specific instructions for the AI agent:

```
Enter your custom instructions for the AI agent.

These could include:
  - Coding standards and conventions
  - Project architecture overview
  - Important files or patterns to follow
  - Testing requirements
  - Any warnings or things to avoid

Type your instructions below. When finished, type 'END' on a new line.
```

### AI Setup Assistant

At the end of installation, an AI agent analyzes your project and automatically configures:

- **`project_prompt.txt`** - Project architecture, coding standards, key directories
- **`build.sh`** - Build verification script for your project type
- **`test.sh`** - Test runner script for your project type

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
AI Setup Assistant
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

The AI agent can analyze your project and automatically configure:
  • project_prompt.txt - Project-specific instructions
  • build.sh - Build verification script
  • test.sh  - Test runner script

Run AI setup assistant now? [Y/n]:
```

The assistant leaves `TASKS.md` for you to fill in with your actual tasks.

## Manual Usage (Advanced)

If you prefer to run commands directly:

```bash
# Run Ralph Loop (from within your project directory)
.ralph/ralph_loop.sh

# With a specific agent
.ralph/ralph_loop.sh auggie
```

The script auto-detects the project directory from its location inside `.ralph/`.

## Directory Structure

### Ralph Loop Repository

The repository is organized into modular components:

```
ralph-loop/                        # This repository
├── install.sh                     # THE entry point - handles everything
├── README.md                      # This documentation
├── lib/                           # Worker scripts (sourced by install.sh)
│   ├── common.sh                  # Shared utilities (colors, prompts)
│   ├── prereqs.sh                 # Prerequisites checking/installation
│   ├── download.sh                # GitHub file downloads
│   ├── detect.sh                  # Project type detection
│   ├── config.sh                  # Config file generation
│   ├── prompts.sh                 # Prompt file generation
│   ├── tasks.sh                   # Task file generation
│   ├── git.sh                     # Git operations
│   └── agent.sh                   # AI agent detection/setup
├── core/                          # Core runtime files (copied to projects)
│   ├── ralph_loop.sh              # Main automation script
│   └── base_prompt.txt            # Global agent instructions
├── templates/                     # Placeholder templates (copied to .ralph/templates/)
│   ├── build.sh                   # Build script template
│   ├── test.sh                    # Test script template
│   ├── platform_prompt.txt        # Platform guidelines template
│   ├── project_prompt.txt         # Project instructions template
│   └── TASKS.md                   # Task list template
└── hooks/                         # Git hooks for development
    └── pre-commit                 # Runs validation before commits
```

### Installed in Your Project

After installation, Ralph Loop files live inside your project:

```
my-project/                        # Your project
├── .ralph/                        # Ralph Loop (all files here)
│   ├── ralph_loop.sh              # Main script
│   ├── base_prompt.txt            # Level 1: Global instructions
│   ├── platform_prompt.txt        # Level 2: Platform guidelines
│   ├── project_prompt.txt         # Level 3: Project-specific instructions
│   ├── config.sh                  # Project settings
│   ├── build.sh                   # Build verification script
│   ├── test.sh                    # Test runner script
│   ├── TASKS.md                   # Task checklist
│   ├── templates/                 # Original templates (for reference)
│   ├── docs/                      # Additional documentation (optional)
│   └── logs/                      # Run logs (auto-created)
└── (your project files)
```

This self-contained structure means you can run `.ralph/ralph_loop.sh` from anywhere in your project.

## 3-Level Prompt System

Instructions are split into three layers that can be edited independently:

| Level | File | Purpose | Examples |
|-------|------|---------|----------|
| 1. Global | `base_prompt.txt` | Ralph Loop workflow instructions | Task format, status markers, one-task-at-a-time rule |
| 2. Platform | `.ralph/platform_prompt.txt` | Platform-specific guidelines | iOS: SwiftUI, MVVM; Python: typing, pytest |
| 3. Project | `.ralph/project_prompt.txt` | Your project's unique requirements | "Uses XcodeGen", "API calls go through NetworkService" |

During installation, placeholder templates are created for platform and project prompts.
The AI setup assistant will configure these files automatically, or you can edit them manually.

**Placeholder detection**: Files containing `<!-- PLACEHOLDER:` are detected as unconfigured
and skipped when building the prompt. This ensures that placeholder content is never sent to agents.

This separation means:
- **Update global rules** without touching project configs
- **Customize platform guidelines** for your specific tech stack
- **Customize project instructions** for your unique requirements

## Test Run Mode

By default, Ralph Loop pauses after completing the first 2 tasks:

```
═══════════════════════════════════════════════════
   🔍 Test Run Checkpoint
═══════════════════════════════════════════════════

The first 2 tasks have been completed.
Please review the changes and verify everything is going according to plan.

You can check:
  • Git log: git log --oneline -2
  • Git diff: git diff HEAD~2
  • Build: run your build command

Continue with the remaining 5 tasks? [y/N]:
```

This gives you a chance to verify the agent is working correctly before letting it continue with more tasks.

**Configure in config.sh:**
```bash
TEST_RUN_ENABLED=true   # Enable/disable checkpoint
TEST_RUN_TASKS=2        # Tasks before checkpoint
```

## Model Selection

At startup, Ralph Loop prompts you to select which AI model to use:

```
Fetching available models for cursor...

Available models (22):

  1) auto
  2) gpt-5.2-codex
  ...
  17) opus-4.5 (current)
  18) sonnet-4.5
  ...

Select model [17]:
```

To skip the prompt, set a default model in `config.sh`:
```bash
DEFAULT_MODEL="opus-4.5"
```

## Progress Indicator

While the agent is working, a real-time progress display shows:

```
⠹ Agent working...
  ⏱  02:34 elapsed  📁  5 files changed
  💬  Implementing the new feature...
```

- **Spinner** - Visual indicator the script is running
- **Elapsed time** - How long the agent has been working
- **Files changed** - Number of modified files (from git)
- **Last output** - Most recent line from the agent's log

When the agent completes, a summary is shown:
```
✓ Agent completed in 2m 34s
  Files changed: 5 | Log lines: 247
```

## Configuration Reference

### config.sh Options

| Variable | Default | Description |
|----------|---------|-------------|
| `PROJECT_NAME` | - | Display name for your project |
| `AGENT_TYPE` | `cursor` | Agent to use: `cursor`, `auggie`, `custom` |
| `DEFAULT_MODEL` | `""` | AI model to use (empty = prompt at startup) |
| `MAX_ITERATIONS` | `50` | Maximum loop iterations |
| `PAUSE_SECONDS` | `5` | Pause between iterations |
| `MAX_CONSECUTIVE_FAILURES` | `3` | Stop after N consecutive failures |
| `TEST_RUN_ENABLED` | `true` | Pause for verification after first N tasks |
| `TEST_RUN_TASKS` | `2` | Number of tasks before checkpoint |
| `REQUIRE_BRANCH` | `true` | Require non-main branch |
| `ALLOWED_BRANCHES` | `""` | Specific allowed branches (empty = any) |
| `AUTO_COMMIT` | `true` | Auto-commit after each task |
| `COMMIT_PREFIX` | `feat` | Commit message prefix |
| `COMMIT_SCOPE` | `""` | Commit scope, e.g., `ios` |
| `BUILD_GATE_ENABLED` | `true` | Verify builds between tasks |
| `BUILD_FIX_ATTEMPTS` | `1` | Attempts to fix broken builds |

### Build and Test Scripts

Ralph Loop uses separate executable scripts for build verification and testing:

**`.ralph/build.sh`** - Required if `BUILD_GATE_ENABLED=true`
```bash
#!/bin/bash
# Example for iOS
xcodebuild -scheme "MyApp" -destination 'platform=iOS Simulator,name=iPhone 16' build
```

**`.ralph/test.sh`** - Optional, for test gates
```bash
#!/bin/bash
# Example for iOS
xcodebuild -scheme "MyApp" -destination 'platform=iOS Simulator,name=iPhone 16' test
```

Both scripts must exit 0 on success and non-zero on failure. The AI setup assistant configures these automatically during installation.

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

### Running with Different Agents

```bash
# Use Cursor (default)
.ralph/ralph_loop.sh

# Use Augment
.ralph/ralph_loop.sh auggie

# Use custom agent (defined in config.sh)
.ralph/ralph_loop.sh custom
```

### Resuming After Interruption

Just run the script again - it picks up from the first unchecked task:

```bash
.ralph/ralph_loop.sh
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

