package mock

import (
	"net/http"
	"sync"
	"time"
)

type RequestRecord struct {
	ID        uint64      `json:"id"`
	Time      time.Time   `json:"time"`
	Method    string      `json:"method"`
	Path      string      `json:"path"`
	Query     string      `json:"query,omitempty"`
	Headers   http.Header `json:"headers"`
	Body      string      `json:"body,omitempty"`
	RouteID   string      `json:"route_id,omitempty"`
	Matched   bool        `json:"matched"`
	Status    int         `json:"status"`
	Duration  int64       `json:"duration_ms"`
	Truncated bool        `json:"truncated,omitempty"`
}

type Recorder struct {
	mu      sync.RWMutex
	nextID  uint64
	records []RequestRecord
	limit   func() int
}

func NewRecorder(limit func() int) *Recorder {
	return &Recorder{limit: limit}
}

func (r *Recorder) Add(record RequestRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	record.ID = r.nextID
	r.records = append(r.records, record)
	limit := r.limit()
	if limit <= 0 {
		limit = 500
	}
	if extra := len(r.records) - limit; extra > 0 {
		copy(r.records, r.records[extra:])
		r.records = r.records[:limit]
	}
}

func (r *Recorder) List(after uint64) []RequestRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]RequestRecord, 0, len(r.records))
	for _, record := range r.records {
		if record.ID > after {
			result = append(result, record)
		}
	}
	return result
}

func (r *Recorder) Clear() {
	r.mu.Lock()
	r.records = nil
	r.mu.Unlock()
}
