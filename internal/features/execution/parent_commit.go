package execution

import (
	"fmt"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func onlyParentCommitHandoff(planned []string) bool {
	if len(planned) == 0 {
		return false
	}
	for _, capability := range planned {
		if !strings.HasPrefix(capability, "git-metadata-write:") {
			return false
		}
	}
	return true
}

// applyParentCommitIfPlanned resolves the narrow Git-index restriction without
// widening the worker sandbox. Other failed capabilities remain ordinary root
// handoffs because a commit receipt cannot prove they were recovered.
func applyParentCommitIfPlanned(w *workspace.Workspace, h store.RootHandoff, planned []string, now time.Time) (store.ParentCommitReceipt, bool, error) {
	if !onlyParentCommitHandoff(planned) || len(h.Unresolved) > 0 {
		return store.ParentCommitReceipt{}, false, nil
	}
	parent, err := agentid.Resolve(w)
	if err != nil {
		return store.ParentCommitReceipt{}, false, err
	}
	if parent.ID != agentid.RootID || parent.Grant != model.GrantRW {
		return store.ParentCommitReceipt{}, false, fmt.Errorf("parent-mediated commit requires root rw authority; current identity is %s (%s)", parent.ID, parent.Grant)
	}
	receipt, err := store.ApplyParentCommit(w, h, now)
	if err != nil {
		return receipt, false, err
	}
	request, err := store.LoadParentCommitRequest(w, h.RunID)
	if err != nil {
		return receipt, false, err
	}
	origin := "parent-commit:" + request.RequestID
	events, err := eventlog.List(w, eventlog.Query{About: request.TaskID, Kinds: []model.EventKind{model.EventCommit}})
	if err != nil {
		return receipt, false, err
	}
	for _, event := range events {
		if event.Origin == origin {
			return receipt, true, nil
		}
	}
	body := fmt.Sprintf("%s %s\nrole: %s\nparent-mediated request: %s", receipt.Commit, request.Message, request.Role, request.RequestID)
	if _, err := eventlog.Append(w, request.ChildID, model.EventCommit, request.TaskID, origin, body); err != nil {
		return receipt, true, fmt.Errorf("record parent-mediated commit event: %w", err)
	}
	return receipt, true, nil
}
