package metrics

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Collect is the one durable-record adapter used by feature slices. Window
// scoping happens before aggregation, so a sample cannot leak in from an older
// scenario merely because it belongs to the same task.
func Collect(w *workspace.Workspace, window Window, project string) Report {
	tasks, _ := store.ListTasks(w, project, "")
	byID := make(map[string]*store.Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}

	entries, _ := os.ReadDir(w.RunsDir())
	runs := make([]Run, 0, len(entries))
	interventionSampled := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := w.RunDir(entry.Name())
		rec, err := procmon.ReadRecord(filepath.Join(dir, "proc.txt"))
		if err != nil || (window.Since != nil && rec.Started.Before(*window.Since)) ||
			(window.Until != nil && !rec.Started.Before(*window.Until)) {
			continue
		}
		task := byID[rec.Task]
		if project != "" && task == nil {
			continue
		}
		run := Run{Task: rec.Task}
		run.Outcome, run.Elapsed = readOutcome(filepath.Join(dir, "outcome.md"))
		if tokens, ok := readInt(filepath.Join(dir, "usage.txt"), "output_tokens"); ok {
			run.OutputTokens = &tokens
		}
		if budget, ok := readInt(filepath.Join(dir, "invocation.txt"), "max_tokens"); ok {
			run.TokenBudget = &budget
		}
		if task != nil && task.Status == model.StatusDone && !interventionSampled[task.ID] {
			intervened := store.LogHasStamp(task, "adopted by a-root")
			run.HumanIntervention = &intervened
			interventionSampled[task.ID] = true
		}
		runs = append(runs, run)
	}
	return Summarise(window, runs)
}

func readOutcome(path string) (string, time.Duration) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", 0
	}
	var outcome string
	var elapsed time.Duration
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "outcome":
			outcome = strings.TrimSpace(value)
		case "elapsed":
			elapsed, _ = time.ParseDuration(strings.TrimSpace(value))
		}
	}
	return outcome, elapsed
}

func readInt(path, want string) (int, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != want {
			continue
		}
		field := strings.Fields(strings.TrimSpace(value))
		if len(field) == 0 {
			return 0, false
		}
		n, err := strconv.Atoi(field[0])
		return n, err == nil && n >= 0
	}
	return 0, false
}
