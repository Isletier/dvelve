package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Task is the unit of work flowing through the pipeline.
type Task struct {
	ID    int64
	Value int
	Slow  bool
}

// Result carries the processed output of a single Task.
type Result struct {
	TaskID  int64
	Input   int
	Output  int // square of Input
	Elapsed time.Duration
	Err     error
}

// Report is a batch of results ready to print.
type Report struct {
	Batch   []Result
	BatchNo int
}

// startGenerators launches cfg.Generators goroutines, each producing Tasks
// into rawCh. The channel is closed when all generators finish.
func startGenerators(ctx context.Context, cfg Config, rng *rand.Rand, rawCh chan<- Task) {
	var wg sync.WaitGroup
	var mu sync.Mutex // guards rng

	nextID := int64(1)

	for i := range cfg.Generators {
		wg.Add(1)
		go func(genID int) {
			defer wg.Done()
			generate(ctx, genID, cfg, rng, &mu, &nextID, rawCh)
		}(i + 1)
	}

	go func() {
		wg.Wait()
		close(rawCh)
	}()
}

// generate is the body of a single generator goroutine.
func generate(ctx context.Context, id int, cfg Config, rng *rand.Rand, mu *sync.Mutex, nextID *int64, out chan<- Task) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		mu.Lock()
		taskID := *nextID
		*nextID++
		value := rng.Intn(1000) + 1
		slow := rng.Float64() < cfg.SlowChance
		mu.Unlock()

		t := Task{ID: taskID, Value: value, Slow: slow}

		select {
		case out <- t:
		case <-ctx.Done():
			return
		}

		// Simulate pacing: generators don't flood the pipeline.
		pause := time.Duration(rng.Intn(50)) * time.Millisecond
		select {
		case <-time.After(pause):
		case <-ctx.Done():
			return
		}
	}
}

// aggregate drains resultCh, batches results, and forwards batches to printCh.
func aggregate(ctx context.Context, resultCh <-chan Result, printCh chan<- Report, stats *Stats) {
	const batchCap = 10

	batch := make([]Result, 0, batchCap)
	batchNo := 0
	flush := func() {
		if len(batch) == 0 {
			return
		}
		batchNo++
		select {
		case printCh <- Report{Batch: batch, BatchNo: batchNo}:
		case <-ctx.Done():
		}
		batch = make([]Result, 0, batchCap)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case r, ok := <-resultCh:
			if !ok {
				flush()
				close(printCh)
				return
			}
			if r.Err != nil {
				stats.AddError()
			} else {
				stats.AddProcessed()
			}
			batch = append(batch, r)
			if len(batch) >= batchCap {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}

// printer reads Report values and writes a summary line for each batch.
func printer(ctx context.Context, printCh <-chan Report) {
	for {
		select {
		case rep, ok := <-printCh:
			if !ok {
				return
			}
			printReport(rep)
		case <-ctx.Done():
			return
		}
	}
}

func printReport(rep Report) {
	var sumElapsed time.Duration
	errCount := 0
	for _, r := range rep.Batch {
		sumElapsed += r.Elapsed
		if r.Err != nil {
			errCount++
		}
	}
	avg := time.Duration(0)
	if len(rep.Batch) > 0 {
		avg = sumElapsed / time.Duration(len(rep.Batch))
	}
	fmt.Printf("batch %4d  items=%d  errors=%d  avgElapsed=%s\n",
		rep.BatchNo, len(rep.Batch), errCount, avg.Round(time.Millisecond))
}
