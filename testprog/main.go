// testprog is a multi-goroutine workload designed for exercising the DVAP
// adapter. It runs a concurrent pipeline:
//
//	Generators → Fan-in → Workers → Aggregator → Printer
//	                                             ↑
//	                                         Monitor (periodic)
//
// Run it under dlv and poke around with breakpoints in the various files to
// see goroutines, threads, and breakpoints reflected in the DVAP SSE stream.
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var (
		generators = flag.Int("generators", 3, "number of generator goroutines")
		workers    = flag.Int("workers", 5, "number of worker goroutines")
		batchSize  = flag.Int("batch", 20, "numbers per generator batch")
		slowChance = flag.Float64("slow", 0.1, "probability a task is 'slow' (0–1)")
	)
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := Config{
		Generators: *generators,
		Workers:    *workers,
		BatchSize:  *batchSize,
		SlowChance: *slowChance,
		Seed:       time.Now().UnixNano(),
	}

	fmt.Fprintf(os.Stderr, "testprog starting  generators=%d workers=%d batch=%d\n",
		cfg.Generators, cfg.Workers, cfg.BatchSize)

	stats := NewStats()
	Run(ctx, cfg, stats)

	fmt.Fprintf(os.Stderr, "\ntestprog done — processed %d items, %d errors\n",
		stats.Processed(), stats.Errors())
}

// Config holds the runtime parameters for the pipeline.
type Config struct {
	Generators int
	Workers    int
	BatchSize  int
	SlowChance float64
	Seed       int64
}

// Run starts all pipeline stages and blocks until ctx is done.
func Run(ctx context.Context, cfg Config, stats *Stats) {
	rng := rand.New(rand.NewSource(cfg.Seed))

	// Stage 1: generators → raw channel.
	rawCh := make(chan Task, cfg.Workers*2)
	startGenerators(ctx, cfg, rng, rawCh)

	// Stage 2: workers → results channel.
	resultCh := make(chan Result, cfg.Workers)
	startWorkers(ctx, cfg, rawCh, resultCh, stats)

	// Stage 3: aggregator collects results and feeds the printer.
	printCh := make(chan Report, 4)
	go aggregate(ctx, resultCh, printCh, stats)

	// Stage 4: printer writes to stderr.
	go printer(ctx, printCh)

	// Stage 5: monitor goroutine — periodic health report.
	go monitor(ctx, stats)

	// Stage 6: probe goroutines — blocked most of the time, useful as
	// breakpoint targets to observe Gwaiting → Grunning transitions.
	probeCh := make(chan struct{}, 1)
	go probe(ctx, "alpha", probeCh)
	go probe(ctx, "beta", probeCh)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case probeCh <- struct{}{}:
			default:
			}
		}
	}
}
