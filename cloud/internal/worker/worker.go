// Package worker implements the bounded background-worker lifecycle.
package worker

import (
	"context"
	"fmt"
	"time"
)

type Job interface {
	Run(context.Context) error
}

type JobFunc func(context.Context) error

func (f JobFunc) Run(ctx context.Context) error { return f(ctx) }

// Run invokes a job immediately and on each interval until cancellation.
// Individual cycles cannot outlive timeout.
func Run(ctx context.Context, interval, timeout time.Duration, job Job, report func(error)) {
	if report == nil {
		report = func(error) {}
	}
	run := func() {
		cycle, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := job.Run(cycle); err != nil {
			report(fmt.Errorf("worker cycle failed: %w", err))
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
