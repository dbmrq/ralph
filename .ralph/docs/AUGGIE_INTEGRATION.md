# Auggie (Augment CLI) Integration Guide

## Overview

Auggie is the Augment CLI tool for agentic coding. This document details how Ralph Go should integrate with Auggie based on official documentation.

**Reference**: https://docs.augmentcode.com/cli/reference

## Installation

```bash
npm install -g @augmentcode/auggie
```

### Detection
```go
func IsAuggieAvailable() bool {
    _, err := exec.LookPath("auggie")
    return err == nil
}
```

## Authentication

Auggie requires authentication. For automation, use session tokens:

### Interactive Login (One-time setup)
```bash
auggie login
```

### Get Session Token
```bash
auggie tokens print
```

### Use Token in Automation
```bash
AUGMENT_SESSION_AUTH='<token>' auggie --print "prompt"
```

## CLI Flags Reference

### Core Execution Flags
| Flag | Description |
|------|-------------|
| `auggie --print` (`-p`) | Output simple text for one instruction and exit |
| `auggie --quiet` | Output only the final response |
| `auggie --compact` | Output tool calls, results, and final response as one line each |
| `auggie -p --output-format json` | Structured JSON output (useful for parsing) |

### Input Methods
| Method | Example |
|--------|---------|
| Direct instruction | `auggie --print "Fix the errors"` |
| Stdin pipe | `cat file \| auggie --print "Summarize"` |
| File input | `auggie --print "Summarize" < file.txt` |
| Instruction file | `auggie --instruction-file /path/to/file.txt` |

### Session Management
| Flag | Description |
|------|-------------|
| `auggie --continue` (`-c`) | Resume previous conversation |
| `auggie --dont-save-session` | Don't save conversation to history |

### Configuration
| Flag | Description |
|------|-------------|
| `auggie --model "name"` | Select model (use names from `auggie models list`) |
| `auggie --workspace-root /path` | Specify workspace root |
| `auggie --rules /path/to/rules.md` | Additional rules to append |

## Model Discovery

Auggie supports model listing:

```bash
auggie models list
```

### Implementation
```go
func (a *AuggieAgent) ListModels() ([]Model, error) {
    output, err := exec.Command("auggie", "models", "list").Output()
    if err != nil {
        return nil, fmt.Errorf("failed to list models: %w", err)
    }
    return parseModelsOutput(output), nil
}
```

## Execution Modes

### Non-Interactive Mode (for Ralph)
```bash
# Standard execution
AUGMENT_SESSION_AUTH='<token>' auggie --print --quiet "prompt"

# With JSON output for parsing
AUGMENT_SESSION_AUTH='<token>' auggie -p --output-format json "prompt"

# Via stdin pipe
cat prompt.txt | AUGMENT_SESSION_AUTH='<token>' auggie --print --quiet
```

### Session Continuation (Pause/Resume)
```bash
# First run saves session
auggie --print "Start implementing feature X"

# Continue previous session
auggie --continue "Continue where you left off"
```

For Ralph, store session state and use `--continue` when resuming paused tasks.

## Implementation

### Basic Execution
```go
func (a *AuggieAgent) Run(ctx context.Context, prompt string, opts RunOptions) (Result, error) {
    args := []string{"--print", "--quiet"}

    if opts.Model != "" {
        args = append(args, "--model", opts.Model)
    }

    // Add instruction
    args = append(args, prompt)

    cmd := exec.CommandContext(ctx, "auggie", args...)
    cmd.Dir = opts.WorkDir
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("AUGMENT_SESSION_AUTH=%s", a.sessionToken))

    // Capture output for real-time streaming
    var stdout, stderr bytes.Buffer
    cmd.Stdout = io.MultiWriter(&stdout, opts.LogWriter)
    cmd.Stderr = &stderr

    startTime := time.Now()
    err := cmd.Run()
    duration := time.Since(startTime)

    return Result{
        Output:    stdout.String(),
        Duration:  duration,
        Status:    parseStatus(stdout.String()),
        ExitCode:  cmd.ProcessState.ExitCode(),
        SessionID: a.extractSessionID(stdout.String()),
    }, err
}
```

### Session Continuation
```go
func (a *AuggieAgent) Continue(ctx context.Context, sessionID string, prompt string, opts RunOptions) (Result, error) {
    args := []string{"--print", "--quiet", "--continue"}

    if opts.Model != "" {
        args = append(args, "--model", opts.Model)
    }

    args = append(args, prompt)

    cmd := exec.CommandContext(ctx, "auggie", args...)
    cmd.Dir = opts.WorkDir
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("AUGMENT_SESSION_AUTH=%s", a.sessionToken))

    // ... rest of execution
}
```

### Session Token Management
```go
type AuggieAgent struct {
    sessionToken string
}

func NewAuggieAgent() (*AuggieAgent, error) {
    // Try environment variable first
    token := os.Getenv("AUGMENT_SESSION_AUTH")
    if token != "" {
        return &AuggieAgent{sessionToken: token}, nil
    }

    // Try to get from auggie tokens print
    output, err := exec.Command("auggie", "tokens", "print").Output()
    if err != nil {
        return nil, fmt.Errorf("auggie not authenticated: run 'auggie login' first")
    }

    token = strings.TrimSpace(string(output))
    return &AuggieAgent{sessionToken: token}, nil
}

func (a *AuggieAgent) CheckAuth() error {
    if a.sessionToken == "" {
        return fmt.Errorf("no session token: run 'auggie login' and 'auggie tokens print'")
    }
    // Optionally verify token is valid
    return nil
}
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AUGMENT_SESSION_AUTH` | Authentication JSON token |
| `AUGMENT_API_URL` | Backend API endpoint (optional) |
| `AUGMENT_API_TOKEN` | Alternative auth token |
| `GITHUB_API_TOKEN` | GitHub API token for integrations |

## Error Handling

### Common Errors
1. **Not authenticated**: Run `auggie login`
2. **Token expired**: Re-authenticate
3. **API unavailable**: Network issues
4. **Rate limited**: Implement backoff

```go
func (a *AuggieAgent) handleError(err error, output string) error {
    if strings.Contains(output, "not authenticated") {
        return &AuthError{Message: "Auggie not authenticated. Run 'auggie login'"}
    }
    if strings.Contains(output, "rate limit") {
        return &RateLimitError{RetryAfter: time.Minute}
    }
    return err
}
```

## Configuration Example

```yaml
# .ralph/config.yaml
agent:
  default: auggie  # or leave empty to prompt when multiple available

timeout:
  active: 2h    # While producing output
  stuck: 30m    # No output threshold

# Auggie uses AUGMENT_SESSION_AUTH from environment
# Set via: export AUGMENT_SESSION_AUTH="$(auggie tokens print)"
```

## Testing the Integration

```go
func TestAuggieAgent(t *testing.T) {
    if !IsAuggieAvailable() {
        t.Skip("Auggie not installed")
    }

    agent, err := NewAuggieAgent()
    if err != nil {
        t.Skipf("Auggie not authenticated: %v", err)
    }

    // Test model listing
    models, err := agent.ListModels()
    require.NoError(t, err)
    assert.NotEmpty(t, models)

    // Test basic execution
    result, err := agent.Run(context.Background(), "Say 'Hello'", RunOptions{
        WorkDir: t.TempDir(),
    })

    require.NoError(t, err)
    assert.Contains(t, result.Output, "Hello")
}
```

