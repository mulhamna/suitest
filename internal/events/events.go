package events

import "time"

// Event describes a structured run event for CLI or web consumers.
type Event struct {
	Time    time.Time      `json:"time"`
	Type    string         `json:"type"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// Emitter receives structured run events.
type Emitter interface {
	Emit(event Event)
}

// NopEmitter ignores all events.
type NopEmitter struct{}

// Emit implements Emitter.
func (NopEmitter) Emit(event Event) {}
