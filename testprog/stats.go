package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Stats is a lock-free counter bag shared across pipeline stages.
type Stats struct {
	processed atomic.Int64
	errors    atomic.Int64
}

func NewStats() *Stats { return &Stats{} }

func (s *Stats) AddProcessed() { s.processed.Add(1) }
func (s *Stats) AddError()     { s.errors.Add(1) }
func (s *Stats) Processed() int64 { return s.processed.Load() }
func (s *Stats) Errors() int64    { return s.errors.Load() }

// monitor is a long-lived goroutine that prints a health summary every few
// seconds. It is always Grunnable/Grunning briefly when it wakes, then parks
// on the ticker — a good example of the filter in action.
func monitor(ctx context.Context, stats *Stats) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(start).Round(time.Second)
			proc := stats.Processed()
			errs := stats.Errors()
			rate := float64(proc) / elapsed.Seconds()
			fmt.Printf("[monitor] uptime=%-8s processed=%-6d errors=%-4d rate=%.1f/s\n",
				elapsed, proc, errs, rate)
		}
	}
}

// probe is a goroutine that normally blocks on probeCh. When it receives a
// signal it does a quick burst of work, then goes back to waiting. This makes
// it alternate between Gwaiting and Grunning — useful for observing how the
// adapter filters goroutines by status.
func probe(ctx context.Context, name string, probeCh <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-probeCh:
			if !ok {
				return
			}
			doProbeWork(name)
		}
	}
}

// doProbeWork is a short CPU burst so the goroutine is Grunning for a
// measurable time. Put a breakpoint here to catch the goroutine mid-work.
func doProbeWork(name string) {
	acc := 0
	for i := range 100_000 {
		acc ^= i
	}
	_ = acc
	fmt.Printf("[probe:%s] fired\n", name)
}
