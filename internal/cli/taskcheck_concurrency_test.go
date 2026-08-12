package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestTaskCheckConcurrentProcessesPreserveDifferentCriteria(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "concurrent-checks")
	run(t, dir, 0, "project", "add", "Concurrent checks", "--slug", "p", "--goal", "Preserve evidence.")
	run(t, dir, 0, "task", "add", "Check independently", "--project", "p", "--accept", "first", "--accept", "second")

	rendezvous := t.TempDir()
	commands := make([]*exec.Cmd, 2)
	outputs := make([]bytes.Buffer, 2)
	for i := range commands {
		commands[i] = exec.Command(bin, "task", "check", "001", "--n", fmt.Sprint(i+1))
		commands[i].Dir = dir
		commands[i].Env = append(os.Environ(), "DACLI_TEST_TASK_CHECK_RENDEZVOUS="+rendezvous)
		commands[i].Stdout = &outputs[i]
		commands[i].Stderr = &outputs[i]
		if err := commands[i].Start(); err != nil {
			t.Fatalf("starting check %d: %v", i+1, err)
		}
	}
	for i, command := range commands {
		if err := command.Wait(); err != nil {
			t.Fatalf("check %d: %v\n%s", i+1, err, outputs[i].String())
		}
	}

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.FindTask(w, "001")
	if err != nil {
		t.Fatal(err)
	}
	boxes := task.Acceptance()
	done := 0
	for _, box := range boxes {
		if box.Done {
			done++
		}
	}
	if done != 2 || len(boxes) != 2 {
		t.Fatalf("persisted acceptance = [%d/%d], want [2/2]", done, len(boxes))
	}
}
