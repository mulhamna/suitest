package jobs

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mulhamna/suitest/internal/events"
)

// Status is the lifecycle state for a run.
type Status string

const (
	StatusQueued      Status = "queued"
	StatusDiscovering Status = "discovering"
	StatusPlanning    Status = "planning"
	StatusExecuting   Status = "executing"
	StatusFixing      Status = "fixing"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
)

// Run stores lightweight run metadata for web consumers.
type Run struct {
	ID          string         `json:"id"`
	ProjectPath string         `json:"project_path"`
	Mode        string         `json:"mode"`
	Provider    string         `json:"provider"`
	Status      Status         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Events      []events.Event `json:"events,omitempty"`
}

// Manager holds in-memory runs for the local web mode.
type Manager struct {
	mu          sync.RWMutex
	runs        map[string]*Run
	subscribers map[string]map[chan events.Event]struct{}
}

// NewManager creates a new in-memory run manager.
func NewManager() *Manager {
	return &Manager{
		runs:        make(map[string]*Run),
		subscribers: make(map[string]map[chan events.Event]struct{}),
	}
}

// Create registers a new run.
func (m *Manager) Create(projectPath, mode, provider string) *Run {
	now := time.Now()
	run := &Run{
		ID:          uuid.NewString(),
		ProjectPath: projectPath,
		Mode:        mode,
		Provider:    provider,
		Status:      StatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[run.ID] = run
	return run
}

// List returns all known runs.
func (m *Manager) List() []*Run {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Run, 0, len(m.runs))
	for _, run := range m.runs {
		clone := *run
		out = append(out, &clone)
	}
	return out
}

// Get returns a run by id.
func (m *Manager) Get(id string) (*Run, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	run, ok := m.runs[id]
	if !ok {
		return nil, false
	}
	clone := *run
	return &clone, true
}

// UpdateStatus updates lifecycle state.
func (m *Manager) UpdateStatus(id string, status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run, ok := m.runs[id]; ok {
		run.Status = status
		run.UpdatedAt = time.Now()
	}
}

// AppendEvent adds a structured event to the run and broadcasts it.
func (m *Manager) AppendEvent(id string, event events.Event) {
	m.mu.Lock()
	if run, ok := m.runs[id]; ok {
		run.Events = append(run.Events, event)
		run.UpdatedAt = time.Now()
	}
	subscribers := m.snapshotSubscribersLocked(id)
	m.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

// Subscribe registers a listener for run events.
func (m *Manager) Subscribe(id string) (<-chan events.Event, func()) {
	ch := make(chan events.Event, 16)

	m.mu.Lock()
	if _, ok := m.subscribers[id]; !ok {
		m.subscribers[id] = make(map[chan events.Event]struct{})
	}
	m.subscribers[id][ch] = struct{}{}
	m.mu.Unlock()

	cancel := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if subscribers, ok := m.subscribers[id]; ok {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(m.subscribers, id)
			}
		}
		close(ch)
	}

	return ch, cancel
}

func (m *Manager) snapshotSubscribersLocked(id string) []chan events.Event {
	subscriberSet, ok := m.subscribers[id]
	if !ok {
		return nil
	}
	out := make([]chan events.Event, 0, len(subscriberSet))
	for subscriber := range subscriberSet {
		out = append(out, subscriber)
	}
	return out
}
