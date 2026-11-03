package process

import (
	"sync"
	"time"
)

type LogEntry struct {
	ID     uint64    `json:"id"`
	Time   time.Time `json:"time"`
	Source string    `json:"source"`
	Line   string    `json:"line"`
}

type LogStore struct {
	mu      sync.RWMutex
	nextID  uint64
	entries []LogEntry
	limit   int
}

func NewLogStore(limit int) *LogStore {
	return &LogStore{limit: limit}
}

func (s *LogStore) Add(source, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.entries = append(s.entries, LogEntry{ID: s.nextID, Time: time.Now().UTC(), Source: source, Line: line})
	if extra := len(s.entries) - s.limit; extra > 0 {
		copy(s.entries, s.entries[extra:])
		s.entries = s.entries[:s.limit]
	}
}

func (s *LogStore) List(after uint64) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]LogEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		if entry.ID > after {
			result = append(result, entry)
		}
	}
	return result
}

func (s *LogStore) Clear() {
	s.mu.Lock()
	s.entries = nil
	s.mu.Unlock()
}
