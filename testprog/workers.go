package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// startWorkers launches cfg.Workers goroutines that drain rawCh and send
// Results into resultCh. resultCh is closed when all workers finish.
func startWorkers(ctx context.Context, cfg Config, rawCh <-chan Task, resultCh chan<- Result, stats *Stats) {
	var wg sync.WaitGroup

	for i := range cfg.Workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			work(ctx, workerID, rawCh, resultCh)
		}(i + 1)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()
}

// work is the body of a single worker goroutine. It reads Tasks, processes
// them, and emits Results. This is a good place to set a breakpoint when you
// want to inspect a Task mid-flight.
func work(ctx context.Context, id int, in <-chan Task, out chan<- Result) {
	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-in:
			if !ok {
				return
			}
			r := process(id, t)
			select {
			case out <- r:
			case <-ctx.Done():
				return
			}
		}
	}
}

// process performs the actual computation for a Task. Slow tasks sleep to
// simulate I/O latency; bad values (multiples of 97) produce an error so the
// error path is exercised regularly.
func process(workerID int, t Task) Result {
	start := time.Now()

	// Artificial error condition — good breakpoint target.
	if t.Value%97 == 0 {
		return Result{
			TaskID:  t.ID,
			Input:   t.Value,
			Err:     fmt.Errorf("worker %d: rejected value %d (multiple of 97)", workerID, t.Value),
			Elapsed: time.Since(start),
		}
	}

	// Slow tasks simulate blocking I/O — they park the goroutine on a timer,
	// which makes them show up as Gwaiting (and thus filtered by the adapter).
	if t.Slow {
		delay := 150*time.Millisecond + time.Duration(t.Value%100)*time.Millisecond
		time.Sleep(delay) // goroutine is Gwaiting here
	} else {
		// Fast path: spin a few thousand iterations to keep the goroutine
		// Grunning long enough to be visible in the adapter output.
		acc := 0
		for i := range t.Value * 1000 {
			acc += i * i
		}
		_ = acc
	}

	output := t.Value * t.Value
	return Result{
		TaskID:  t.ID,
		Input:   t.Value,
		Output:  output,
		Elapsed: time.Since(start),
	}
}
