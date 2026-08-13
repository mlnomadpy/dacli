package queues

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

type queueReceipt struct {
	Key, Actor, Outcome, Reason, State string
	BeforeCursor                       int
}

var queueReceiptWrite = writeQueueReceipt

func queueTransitionPath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("%x.json", sha256.Sum256([]byte(key))))
}

func applyQueueTransition(w *workspace.Workspace, actor, slug, key, outcome, reason string) (*store.Queue, bool, error) {
	receipts := filepath.Join(w.QueuesDir(), slug+".transitions")
	var q *store.Queue
	var replay bool
	err := store.WithFileLock(filepath.Join(w.QueuesDir(), slug+".transition.lock"), func() error {
		var err error
		q, replay, err = applyLockedQueueTransition(w, actor, slug, key, outcome, reason, receipts)
		return err
	})
	return q, replay, err
}

func applyLockedQueueTransition(w *workspace.Workspace, actor, slug, key, outcome, reason, receipts string) (*store.Queue, bool, error) {
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
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return mdstore.WriteBytes(path, b, 0o600)
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
