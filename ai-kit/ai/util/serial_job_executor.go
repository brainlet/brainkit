// Ported from: packages/ai/src/util/serial-job-executor.ts
package util

import "sync"

// SerialJobExecutor executes jobs serially, one at a time, in the order they are submitted.
type SerialJobExecutor struct {
	mu         sync.Mutex
	queue      []jobEntry
	processing bool
}

type jobEntry struct {
	job    Job
	result chan error
}

// NewSerialJobExecutor creates a new SerialJobExecutor.
func NewSerialJobExecutor() *SerialJobExecutor {
	return &SerialJobExecutor{}
}

// Run queues a job for serial execution and waits for it to complete.
// Returns the error from the job, if any.
func (e *SerialJobExecutor) Run(job Job) error {
	result := make(chan error, 1)

	e.mu.Lock()
	e.queue = append(e.queue, jobEntry{job: job, result: result})
	if !e.processing {
		e.processing = true
		go e.processQueue()
	}
	e.mu.Unlock()

	return <-result
}

func (e *SerialJobExecutor) processQueue() {
	for {
		e.mu.Lock()
		if len(e.queue) == 0 {
			e.processing = false
			e.mu.Unlock()
			return
		}
		entry := e.queue[0]
		e.queue = e.queue[1:]
		e.mu.Unlock()

		err := entry.job()
		entry.result <- err
	}
}
