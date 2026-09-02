package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunIsImmediateBoundedAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		Run(ctx, 5*time.Millisecond, 10*time.Millisecond, JobFunc(func(cycle context.Context) error {
			calls.Add(1)
			<-cycle.Done()
			return nil
		}), nil)
		close(done)
	}()
	time.Sleep(25 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	if calls.Load() == 0 {
		t.Fatal("worker did not run immediately")
	}
}
