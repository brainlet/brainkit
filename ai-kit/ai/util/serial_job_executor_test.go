// Ported from: packages/ai/src/util/serial-job-executor.test.ts
package util

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestSerialJobExecutor_SingleJob(t *testing.T) {
	executor := NewSerialJobExecutor()
	var done bool

	err := executor.Run(func() error {
		done = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("job did not execute")
	}
}

func TestSerialJobExecutor_MultipleJobsInOrder(t *testing.T) {
	executor := NewSerialJobExecutor()
	var executionOrder []int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = executor.Run(func() error {
				mu.Lock()
				executionOrder = append(executionOrder, idx)
				mu.Unlock()
				return nil
			})
		}(i)
	}

	wg.Wait()

	// All 3 should have executed
	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(executionOrder))
	}
}

func TestSerialJobExecutor_HandleErrors(t *testing.T) {
	executor := NewSerialJobExecutor()
	testErr := errors.New("test error")

	err := executor.Run(func() error {
		return testErr
	})
	if !errors.Is(err, testErr) {
		t.Fatalf("expected test error, got %v", err)
	}
}

func TestSerialJobExecutor_OneAtATime(t *testing.T) {
	executor := NewSerialJobExecutor()
	var concurrentJobs int32
	var maxConcurrentJobs int32

	ch1 := make(chan struct{})
	ch2 := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_ = executor.Run(func() error {
			atomic.AddInt32(&concurrentJobs, 1)
			current := atomic.LoadInt32(&concurrentJobs)
			if current > atomic.LoadInt32(&maxConcurrentJobs) {
				atomic.StoreInt32(&maxConcurrentJobs, current)
			}
			<-ch1
			atomic.AddInt32(&concurrentJobs, -1)
			return nil
		})
	}()

	go func() {
		defer wg.Done()
		_ = executor.Run(func() error {
			atomic.AddInt32(&concurrentJobs, 1)
			current := atomic.LoadInt32(&concurrentJobs)
			if current > atomic.LoadInt32(&maxConcurrentJobs) {
				atomic.StoreInt32(&maxConcurrentJobs, current)
			}
			<-ch2
			atomic.AddInt32(&concurrentJobs, -1)
			return nil
		})
	}()

	close(ch1)
	close(ch2)
	wg.Wait()

	if atomic.LoadInt32(&maxConcurrentJobs) > 1 {
		t.Fatal("jobs ran concurrently, expected serial execution")
	}
}

func TestSerialJobExecutor_MixedSuccessAndFailure(t *testing.T) {
	executor := NewSerialJobExecutor()
	var results []string
	var mu sync.Mutex
	testErr := errors.New("test error")

	var wg sync.WaitGroup
	wg.Add(3)

	errs := make([]error, 3)

	go func() {
		defer wg.Done()
		errs[0] = executor.Run(func() error {
			mu.Lock()
			results = append(results, "job1")
			mu.Unlock()
			return nil
		})
	}()

	go func() {
		defer wg.Done()
		errs[1] = executor.Run(func() error {
			return testErr
		})
	}()

	go func() {
		defer wg.Done()
		errs[2] = executor.Run(func() error {
			mu.Lock()
			results = append(results, "job3")
			mu.Unlock()
			return nil
		})
	}()

	wg.Wait()

	// At least one error should be the test error
	foundErr := false
	for _, e := range errs {
		if errors.Is(e, testErr) {
			foundErr = true
		}
	}
	if !foundErr {
		t.Fatal("expected test error in results")
	}
}
