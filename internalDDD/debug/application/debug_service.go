package application

import (
	"sort"
	"sync"
	"time"
)

type DebugError struct {
	Time      time.Time         `json:"time"`
	Level     string            `json:"level"`
	Component string            `json:"component"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
}

type ErrorStore struct {
	mu     sync.RWMutex
	errors []DebugError
	limit  int
}

func NewErrorStore(limit int) *ErrorStore {
	if limit <= 0 {
		limit = 500
	}

	return &ErrorStore{
		errors: make([]DebugError, 0, limit),
		limit:  limit,
	}
}

func (s *ErrorStore) Add(
	level string,
	component string,
	message string,
	details map[string]string,
) {
	entry := DebugError{
		Time:      time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
		Details:   cloneDetails(details),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.errors = append(s.errors, entry)

	if len(s.errors) > s.limit {
		excess := len(s.errors) - s.limit
		s.errors = append([]DebugError(nil), s.errors[excess:]...)
	}
}

func (s *ErrorStore) Error(
	component string,
	message string,
	details map[string]string,
) {
	s.Add("error", component, message, details)
}

func (s *ErrorStore) Warn(
	component string,
	message string,
	details map[string]string,
) {
	s.Add("warn", component, message, details)
}

func (s *ErrorStore) List(limit int) []DebugError {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.errors) {
		limit = len(s.errors)
	}

	result := make([]DebugError, 0, limit)

	for i := len(s.errors) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, cloneError(s.errors[i]))
	}

	return result
}

func (s *ErrorStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.errors)
}

func cloneError(value DebugError) DebugError {
	value.Details = cloneDetails(value.Details)
	return value
}

func cloneDetails(
	details map[string]string,
) map[string]string {
	if len(details) == 0 {
		return nil
	}

	result := make(map[string]string, len(details))

	for key, value := range details {
		result[key] = value
	}

	return result
}

func SortErrorsByTime(errors []DebugError) {
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Time.After(errors[j].Time)
	})
}