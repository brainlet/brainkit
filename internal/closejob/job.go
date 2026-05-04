package closejob

import (
	"context"
	"sync"
)

// Job runs one close operation at a time and lets callers wait with their own
// context. A timed-out waiter does not lose the underlying close job.
type Job struct {
	mu      sync.Mutex
	done    chan struct{}
	err     error
	running bool
}

// Start starts fn unless a close job is already running. It returns the done
// channel for the active job.
func (j *Job) Start(fn func() error) <-chan struct{} {
	j.mu.Lock()
	if j.running {
		done := j.done
		j.mu.Unlock()
		return done
	}
	done := make(chan struct{})
	j.done = done
	j.err = nil
	j.running = true
	j.mu.Unlock()

	go func() {
		err := fn()
		j.mu.Lock()
		if j.done == done {
			j.err = err
			j.running = false
			close(done)
		}
		j.mu.Unlock()
	}()
	return done
}

// Wait waits for done or ctx cancellation. If the job completed, its error is
// returned and the job slot is cleared.
func (j *Job) Wait(ctx context.Context, done <-chan struct{}) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		return j.finish(done)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Running reports whether a close job is still active.
func (j *Job) Running() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.running
}

func (j *Job) finish(done <-chan struct{}) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.done != done {
		return nil
	}
	err := j.err
	if !j.running {
		j.done = nil
		j.err = nil
	}
	return err
}
