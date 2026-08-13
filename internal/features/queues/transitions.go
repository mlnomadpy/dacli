package queues

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

type queueReceipt struct {
	Key, Actor, Outcome, Reason, State string
	BeforeCursor                       int
}

var queueTransitionLocks sync.Map
var queueReceiptWrite = writeQueueReceipt

func queueLock(path string) *sync.Mutex {
	v, _ := queueTransitionLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func queueTransitionPath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("%x.json", sha256.Sum256([]byte(key))))
}

func applyQueueTransition(w *workspace.Workspace, actor, slug, key, outcome, reason string) (*store.Queue, bool, error) {
	receipts := filepath.Join(w.QueuesDir(), slug+".transitions")
	mu := queueLock(filepath.Join(w.QueuesDir(), slug))
	mu.Lock()
	defer mu.Unlock()
	release, err := acquireQueueFileLock(filepath.Join(w.QueuesDir(), slug+".transition.lock"))
	if err != nil {
		return nil, false, err
	}
	defer release()

	q, err := store.LoadQueue(w, slug)
	if err != nil {
		return nil, false, err
	}
	path := queueTransitionPath(receipts, key)
	r, exists, err := readQueueReceipt(path)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		r = queueReceipt{Key: key, Actor: actor, Outcome: outcome, Reason: reason, State: "pending", BeforeCursor: q.Cursor}
		if err := queueReceiptWrite(path, r); err != nil {
			return nil, false, err
		}
	} else if r.Actor != actor || r.Outcome != outcome || r.Reason != reason {
		return nil, false, fmt.Errorf("transition key %q was already used with different attributes", key)
	}
	replay := r.State == "applied"
	if r.State == "pending" {
		switch outcome {
		case "success":
			if q.Cursor == r.BeforeCursor {
				if err := store.Advance(q, ""); err != nil {
					return nil, false, err
				}
			} else if q.Cursor != r.BeforeCursor+1 {
				return nil, false, fmt.Errorf("cannot reconcile transition %q: cursor is %d, expected %d or %d", key, q.Cursor, r.BeforeCursor, r.BeforeCursor+1)
			}
		case "terminal":
			if q.Halted == "" {
				if err := store.Advance(q, reason); err != nil {
					return nil, false, err
				}
			} else if q.Halted != reason {
				return nil, false, fmt.Errorf("cannot reconcile transition %q: queue halted for %q", key, q.Halted)
			}
		case "retryable":
		default:
			return nil, false, fmt.Errorf("unknown queue transition outcome %q", outcome)
		}
		r.State = "applied"
		if err := queueReceiptWrite(path, r); err != nil {
			return nil, false, err
		}
	}
	if outcome == "terminal" {
		deadPath := queueTransitionPath(filepath.Join(w.QueuesDir(), slug+".dead-letter"), key)
		if _, err := os.Stat(deadPath); os.IsNotExist(err) {
			if err := queueReceiptWrite(deadPath, r); err != nil {
				return nil, false, err
			}
		} else if err != nil {
			return nil, false, err
		}
	}
	body := fmt.Sprintf("queue transition key=%q outcome=%s cursor=%d", key, outcome, q.Cursor)
	if reason != "" {
		body += fmt.Sprintf(" reason=%q", reason)
	}
	if err := ensureQueueAudit(w, actor, "q-"+slug, body); err != nil {
		return nil, false, err
	}
	return q, replay, nil
}

func acquireQueueFileLock(path string) (func(), error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_, err = fmt.Fprintf(f, "%d\n", os.Getpid())
			if closeErr := f.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				_ = os.Remove(path)
				return nil, err
			}
			return func() { _ = os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		b, readErr := os.ReadFile(path)
		var pid int
		if readErr == nil {
			_, _ = fmt.Sscanf(string(b), "%d", &pid)
		}
		alive := false
		if pid > 0 {
			if p, findErr := os.FindProcess(pid); findErr == nil {
				alive = p.Signal(syscall.Signal(0)) == nil
			}
		}
		if !alive {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for queue transition lock %s", filepath.Base(path))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func readQueueReceipt(path string) (queueReceipt, bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return queueReceipt{}, false, nil
	}
	if err != nil {
		return queueReceipt{}, false, err
	}
	var r queueReceipt
	err = json.Unmarshal(b, &r)
	return r, true, err
}

func writeQueueReceipt(path string, r queueReceipt) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".transition-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(b); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

func ensureQueueAudit(w *workspace.Workspace, actor, about, body string) error {
	events, err := eventlog.List(w, eventlog.Query{About: about, Kinds: []model.EventKind{model.EventRun}})
	if err != nil {
		return err
	}
	for _, e := range events {
		if e.Actor == actor && strings.TrimSpace(e.Body) == body {
			return nil
		}
	}
	_, err = eventlog.Append(w, actor, model.EventRun, about, "agent", body)
	return err
}
