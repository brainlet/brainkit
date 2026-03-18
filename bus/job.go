package bus

import (
	"sync"
	"time"
)

// Job tracks a cascade of messages sharing a TraceID.
type Job struct {
	TraceID     string    `json:"traceId"`
	Status      string    `json:"status"` // running | completed | failed | timeout
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Messages    int       `json:"messages"`
	Pending     int       `json:"pending"`
}

type jobTracker struct {
	mu        sync.Mutex
	jobs      map[string]*Job
	timeout   time.Duration
	retention time.Duration
	done      chan struct{}
}

func newJobTracker(timeout, retention time.Duration) *jobTracker {
	jt := &jobTracker{
		jobs:      make(map[string]*Job),
		timeout:   timeout,
		retention: retention,
		done:      make(chan struct{}),
	}
	go jt.evictionLoop()
	return jt
}

func (jt *jobTracker) close() {
	close(jt.done)
}

func (jt *jobTracker) getOrCreate(traceID string) *Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	if j, ok := jt.jobs[traceID]; ok {
		return j
	}
	j := &Job{
		TraceID:   traceID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	jt.jobs[traceID] = j
	return j
}

func (jt *jobTracker) get(traceID string) *Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	return jt.jobs[traceID]
}

func (jt *jobTracker) incrementMessages(traceID string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	if j, ok := jt.jobs[traceID]; ok {
		j.Messages++
	}
}

func (jt *jobTracker) incrementPending(traceID string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	if j, ok := jt.jobs[traceID]; ok {
		j.Pending++
	}
}

func (jt *jobTracker) decrementPending(traceID string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	if j, ok := jt.jobs[traceID]; ok {
		j.Pending--
		if j.Pending <= 0 && j.Status == "running" {
			j.Status = "completed"
			j.CompletedAt = time.Now()
		}
	}
}

func (jt *jobTracker) list() []Job {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	jobs := make([]Job, 0, len(jt.jobs))
	for _, j := range jt.jobs {
		jobs = append(jobs, *j)
	}
	return jobs
}

func (jt *jobTracker) evictionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			jt.evict()
		case <-jt.done:
			return
		}
	}
}

func (jt *jobTracker) evict() {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	now := time.Now()

	for id, j := range jt.jobs {
		// Timeout running jobs
		if j.Status == "running" && now.Sub(j.StartedAt) > jt.timeout {
			j.Status = "timeout"
			j.CompletedAt = now
		}
		// Evict completed/timed-out jobs past retention
		if j.Status != "running" && now.Sub(j.CompletedAt) > jt.retention {
			delete(jt.jobs, id)
		}
	}
}
