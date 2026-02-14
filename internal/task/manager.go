// Package task provides task data model and management for ralph.
package task

import (
	"fmt"
	"sort"
	"sync"
)

// Manager provides task state management operations.
// It wraps a Store and provides higher-level operations for task execution.
type Manager struct {
	store *Store
	mu    sync.RWMutex
}

// NewManager creates a new Manager wrapping the given Store.
func NewManager(store *Store) *Manager {
	return &Manager{
		store: store,
	}
}

// Store returns the underlying store.
func (m *Manager) Store() *Store {
	return m.store
}

// Load loads tasks from the store file.
func (m *Manager) Load() error {
	return m.store.Load()
}

// Save persists tasks to the store file.
func (m *Manager) Save() error {
	return m.store.Save()
}

// GetNext returns the next task to be executed.
// It returns tasks in order of priority:
// 1. In-progress tasks (to resume)
// 2. Paused tasks (to resume)
// 3. Pending tasks (by order)
// Returns nil if no tasks are available.
func (m *Manager) GetNext() *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := m.store.Tasks()

	// Sort by order
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Order < tasks[j].Order
	})

	// First, check for in-progress tasks (should resume)
	for _, t := range tasks {
		if t.Status == StatusInProgress {
			return t
		}
	}

	// Next, check for paused tasks
	for _, t := range tasks {
		if t.Status == StatusPaused {
			return t
		}
	}

	// Finally, get the first pending task
	for _, t := range tasks {
		if t.Status == StatusPending {
			return t
		}
	}

	return nil
}

// GetByID returns a task by its ID.
func (m *Manager) GetByID(id string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.Get(id)
}

// MarkComplete marks a task as completed.
// Returns an error if the task doesn't exist.
func (m *Manager) MarkComplete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.store.Get(id)
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}

	task.MarkCompleted()
	return m.store.Update(task)
}

// Skip marks a task as skipped.
// Returns an error if the task doesn't exist.
func (m *Manager) Skip(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.store.Get(id)
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}

	task.MarkSkipped()
	return m.store.Update(task)
}

// Pause marks a task as paused for later resumption.
// Returns an error if the task doesn't exist.
func (m *Manager) Pause(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.store.Get(id)
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}

	task.MarkPaused()
	return m.store.Update(task)
}

// MarkFailed marks a task as failed.
// Returns an error if the task doesn't exist.
func (m *Manager) MarkFailed(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.store.Get(id)
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}

	task.MarkFailed()
	return m.store.Update(task)
}

// CountRemaining returns the number of tasks that are not in a terminal state.
// This includes pending, in_progress, and paused tasks.
func (m *Manager) CountRemaining() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, t := range m.store.Tasks() {
		if !t.Status.IsTerminal() {
			count++
		}
	}
	return count
}

// CountCompleted returns the number of completed tasks.
func (m *Manager) CountCompleted() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.CountByStatus(StatusCompleted)
}

// CountTotal returns the total number of tasks.
func (m *Manager) CountTotal() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.Count()
}

// HasRemaining returns true if there are tasks remaining to be completed.
func (m *Manager) HasRemaining() bool {
	return m.CountRemaining() > 0
}

// All returns all tasks sorted by order.
func (m *Manager) All() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := m.store.Tasks()
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Order < tasks[j].Order
	})
	return tasks
}

// StartIteration starts a new iteration for the given task.
// Returns the iteration and an error if the task doesn't exist.
func (m *Manager) StartIteration(id string) (*Iteration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("task %q not found", id)
	}

	iter := task.StartIteration()
	if err := m.store.Update(task); err != nil {
		return nil, err
	}

	return iter, nil
}

// EndIteration ends the current iteration for the given task.
// Returns an error if the task doesn't exist or has no active iteration.
func (m *Manager) EndIteration(id, result, output, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.store.Get(id)
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}

	if err := task.EndIteration(result, output, sessionID); err != nil {
		return err
	}

	return m.store.Update(task)
}

// UpdateTask updates a task in the store.
// Returns an error if the task doesn't exist.
func (m *Manager) UpdateTask(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Update(task)
}

// AddTask adds a new task to the store.
// Returns an error if a task with the same ID already exists.
func (m *Manager) AddTask(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Add(task)
}

// Progress returns the completion progress as (completed, total).
func (m *Manager) Progress() (completed, total int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := m.store.Tasks()
	total = len(tasks)
	for _, t := range tasks {
		if t.Status == StatusCompleted {
			completed++
		}
	}
	return
}

// GetByStatus returns all tasks with the given status.
func (m *Manager) GetByStatus(status TaskStatus) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.GetByStatus(status)
}

// Resume resumes a paused task by starting a new iteration.
// Returns the iteration and an error if the task doesn't exist or isn't paused.
func (m *Manager) Resume(id string) (*Iteration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.store.Get(id)
	if !ok {
		return nil, fmt.Errorf("task %q not found", id)
	}

	if task.Status != StatusPaused {
		return nil, fmt.Errorf("task %q is not paused (status: %s)", id, task.Status)
	}

	iter := task.Resume()
	if err := m.store.Update(task); err != nil {
		return nil, err
	}

	return iter, nil
}

// ReorderTasks updates the order of tasks based on the provided ID order.
// Tasks not in the list retain their relative ordering after listed tasks.
func (m *Manager) ReorderTasks(orderedIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks := m.store.Tasks()
	taskMap := make(map[string]*Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	// Assign new order based on position in orderedIDs
	order := 1
	for _, id := range orderedIDs {
		if task, ok := taskMap[id]; ok {
			task.Order = order
			if err := m.store.Update(task); err != nil {
				return fmt.Errorf("failed to update task %q: %w", id, err)
			}
			order++
		}
	}

	// Handle tasks not in the list
	for _, t := range tasks {
		found := false
		for _, id := range orderedIDs {
			if t.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Order = order
			if err := m.store.Update(t); err != nil {
				return fmt.Errorf("failed to update task %q: %w", t.ID, err)
			}
			order++
		}
	}

	return nil
}
