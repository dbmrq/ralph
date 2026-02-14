// Package task provides task data model and management for ralph.
package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultStoreFilename is the default filename for task storage.
const DefaultStoreFilename = "tasks.json"

// StoreMetadata contains information about the task store itself.
type StoreMetadata struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskStore represents the JSON storage for tasks.
type TaskStore struct {
	Metadata StoreMetadata `json:"metadata"`
	Tasks    []*Task       `json:"tasks"`
}

// Store handles reading and writing tasks to JSON storage.
type Store struct {
	path  string
	mu    sync.RWMutex
	store *TaskStore
}

// NewStore creates a new Store instance for the given path.
// It does not load or create the file; call Load() or Save() for that.
func NewStore(path string) *Store {
	return &Store{
		path: path,
		store: &TaskStore{
			Metadata: StoreMetadata{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Tasks: []*Task{},
		},
	}
}

// NewStoreInDir creates a Store for the default tasks.json in the given directory.
func NewStoreInDir(dir string) *Store {
	return NewStore(filepath.Join(dir, DefaultStoreFilename))
}

// Path returns the file path of the store.
func (s *Store) Path() string {
	return s.path
}

// Load reads tasks from the JSON file.
// If the file doesn't exist, initializes an empty store.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize empty store
			s.store = &TaskStore{
				Metadata: StoreMetadata{
					Version:   "1.0",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Tasks: []*Task{},
			}
			return nil
		}
		return fmt.Errorf("failed to read task store: %w", err)
	}

	var store TaskStore
	if err := json.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("failed to parse task store: %w", err)
	}

	s.store = &store
	return nil
}

// Save writes tasks to the JSON file.
// Creates parent directories if they don't exist.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store.Metadata.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(s.store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task store: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write task store: %w", err)
	}

	return nil
}

// Tasks returns a copy of all tasks.
func (s *Store) Tasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, len(s.store.Tasks))
	for i, t := range s.store.Tasks {
		tasks[i] = t.Clone()
	}
	return tasks
}

// Count returns the total number of tasks.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.store.Tasks)
}

// Get retrieves a task by ID.
func (s *Store) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.store.Tasks {
		if t.ID == id {
			return t.Clone(), true
		}
	}
	return nil, false
}

// Add adds a new task to the store.
// Returns an error if a task with the same ID already exists.
func (s *Store) Add(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.store.Tasks {
		if t.ID == task.ID {
			return fmt.Errorf("task with ID %q already exists", task.ID)
		}
	}

	clone := task.Clone()
	// Set order to be after all existing tasks
	if clone.Order == 0 && len(s.store.Tasks) > 0 {
		maxOrder := 0
		for _, t := range s.store.Tasks {
			if t.Order > maxOrder {
				maxOrder = t.Order
			}
		}
		clone.Order = maxOrder + 1
	}

	s.store.Tasks = append(s.store.Tasks, clone)
	return nil
}

// AddAll adds multiple tasks to the store.
// Returns an error if any task with the same ID already exists.
func (s *Store) AddAll(tasks []*Task) error {
	for _, task := range tasks {
		if err := s.Add(task); err != nil {
			return err
		}
	}
	return nil
}

// Update updates an existing task in the store.
// Returns an error if the task doesn't exist.
func (s *Store) Update(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.store.Tasks {
		if t.ID == task.ID {
			s.store.Tasks[i] = task.Clone()
			return nil
		}
	}
	return fmt.Errorf("task with ID %q not found", task.ID)
}

// Delete removes a task from the store by ID.
// Returns an error if the task doesn't exist.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.store.Tasks {
		if t.ID == id {
			s.store.Tasks = append(s.store.Tasks[:i], s.store.Tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("task with ID %q not found", id)
}

// Clear removes all tasks from the store.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store.Tasks = []*Task{}
}

// Exists checks if a task with the given ID exists.
func (s *Store) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.store.Tasks {
		if t.ID == id {
			return true
		}
	}
	return false
}

// CountByStatus returns the number of tasks with the given status.
func (s *Store) CountByStatus(status TaskStatus) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, t := range s.store.Tasks {
		if t.Status == status {
			count++
		}
	}
	return count
}

// GetByStatus returns all tasks with the given status.
func (s *Store) GetByStatus(status TaskStatus) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*Task
	for _, t := range s.store.Tasks {
		if t.Status == status {
			tasks = append(tasks, t.Clone())
		}
	}
	return tasks
}

// SetTasks replaces all tasks with the provided list.
// This is useful for bulk imports.
func (s *Store) SetTasks(tasks []*Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store.Tasks = make([]*Task, len(tasks))
	for i, t := range tasks {
		s.store.Tasks[i] = t.Clone()
	}
}

// Metadata returns the store metadata.
func (s *Store) Metadata() StoreMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.Metadata
}
