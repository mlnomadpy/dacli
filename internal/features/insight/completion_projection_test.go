package insight

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
)

func TestStatusSeparatesImplementedUnlandedFromDone(t *testing.T) {
	w, ctx := doctorEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Await landing", store.TaskOpts{Accept: []string{"verified"}})
	if err != nil {
		t.Fatal(err)
	}
	task.Doc.Front.Set("completion_state", "implemented-unlanded")
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	ctx.Stdout = &bytes.Buffer{}
	if err := cmdStatus(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	got := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(got, "open:1") || !strings.Contains(got, "done:0") || !strings.Contains(got, "implemented-unlanded:1") {
		t.Fatalf("status conflated implementation with completion: %q", got)
	}
}
