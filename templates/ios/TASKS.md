# Task List

**Purpose:** Atomic tasks for automated agent completion
**Format:** `- [ ] TASK-ID: Description` (unchecked) or `- [x] TASK-ID: Description` (done)

---

## 🧪 Validation Task

- [ ] TEST-001: Verify the project builds and tests pass
  > Goal: Run the build and test commands to ensure the project is in a working state
  > This validates the Ralph Loop setup is working correctly

---

## 📋 Phase 1: [Your First Phase]

### 1.1 [Feature Group Name]

- [ ] FEAT-001: [First task description]
  > Goal: [What this task should accomplish]
  > Reference: [Link to design doc, API spec, or example]
  > Notes: [Any additional context]

- [ ] FEAT-002: [Second task description]
  > Goal: [What this task should accomplish]
  > Reference: [Link to reference]

---

## 📋 Phase 2: [Your Second Phase]

### 2.1 [Feature Group Name]

- [ ] FEAT-010: [Task description]
  > Goal: [What this task should accomplish]

---

## Task Writing Guidelines

When adding tasks:

1. **One atomic change per task** - Each task should be completable in one agent run
2. **Clear success criteria** - The agent should know when the task is done
3. **Include references** - Link to designs, specs, or example implementations
4. **Order matters** - Tasks are completed in order, so dependencies should come first
5. **Use consistent IDs** - Format: `PREFIX-###` (e.g., AUTH-001, UI-015)

Example of a well-written task:
```
- [ ] AUTH-003: Add password validation to signup form
  > Goal: Validate password meets requirements (8+ chars, 1 uppercase, 1 number)
  > Reference: See login form validation in Sources/Auth/LoginViewModel.swift
  > Notes: Show inline error messages, disable submit until valid
```

