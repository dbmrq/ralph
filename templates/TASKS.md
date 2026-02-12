# Task List

**Purpose:** Atomic tasks for automated agent completion
**Format:** `- [ ] TASK-ID: Description` (unchecked) or `- [x] TASK-ID: Description` (done)

---

## 🧪 Validation Task

- [ ] TEST-001: Verify the project builds successfully
  > Goal: Run the build command and ensure it passes
  > This validates the Ralph Loop setup is working correctly

---

## 📋 Your Tasks

<!-- Add your tasks here -->

- [ ] TASK-001: Your first task
  > Goal: Describe what this task should accomplish
  > Reference: Link to any relevant documentation

- [ ] TASK-002: Your second task
  > Goal: Describe what this task should accomplish

---

## Task Writing Tips

1. **One atomic change per task** - Completable in one agent run
2. **Clear success criteria** - Agent knows when it's done
3. **Include references** - Links to designs, specs, examples
4. **Order matters** - Dependencies come first
5. **Use consistent IDs** - Format: `PREFIX-###` (e.g., FEAT-001, AUTH-015)

Example of a well-written task:
```
- [ ] AUTH-003: Add password validation to signup form
  > Goal: Validate password meets requirements (8+ chars, 1 uppercase, 1 number)
  > Reference: See login form validation in Sources/Auth/LoginViewModel.swift
  > Notes: Show inline error messages, disable submit until valid
```

