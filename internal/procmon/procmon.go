// Package procmon watches and terminates spawned-agent process TREES. A
// dacli-spawned agent (e.g. `claude -p ...`) forks a whole tree of helper
// subprocesses; left unsupervised, a hung or runaway agent leaks that tree's
// RAM/CPU/GPU long after dacli has moved on. procmon gives every spawn a
// killable process GROUP and a live resource probe, so a *separate* dacli
// invocation (`dacli agents`, `dacli kill`) can see the tree and reap it as a
// unit — never leaving orphaned children behind.
//
// This is a shared entity, not a feature slice: the execution slice writes the
// records at spawn time; the monitoring/kill commands read them back. Liveness
// is NEVER trusted from the file — it is always probed live.
package procmon

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Record is written to a run's proc.txt at spawn time so a different dacli
// process can find the live tree. PID is the group leader (the runtime binary
// dacli exec'd); PGID equals it, because Setpgid makes that child a new group
// leader. Every subprocess the agent forks inherits PGID unless it detaches.
type Record struct {
	RunID   string
	Child   string
	Task    string
	Role    string
	Runtime string
	PID     int
	PGID    int
	Started time.Time
}

// WriteRecord persists r as key: value lines, matching the run dir's other
// plain-text records (invocation.txt, outcome.md).
func WriteRecord(path string, r Record) error {
	var b strings.Builder
	fmt.Fprintf(&b, "run: %s\n", r.RunID)
	fmt.Fprintf(&b, "child: %s\n", r.Child)
	fmt.Fprintf(&b, "task: %s\n", r.Task)
	fmt.Fprintf(&b, "role: %s\n", r.Role)
	fmt.Fprintf(&b, "runtime: %s\n", r.Runtime)
	fmt.Fprintf(&b, "pid: %d\n", r.PID)
	fmt.Fprintf(&b, "pgid: %d\n", r.PGID)
	fmt.Fprintf(&b, "started: %s\n", r.Started.UTC().Format(time.RFC3339))
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// ReadRecord parses a proc.txt. A missing or malformed file is an error; the
// caller skips it rather than treating it as a live agent.
func ReadRecord(path string) (Record, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	r := Record{}
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "run":
			r.RunID = v
		case "child":
			r.Child = v
		case "task":
			r.Task = v
		case "role":
			r.Role = v
		case "runtime":
			r.Runtime = v
		case "pid":
			r.PID, _ = strconv.Atoi(v)
		case "pgid":
			r.PGID, _ = strconv.Atoi(v)
		case "started":
			r.Started, _ = time.Parse(time.RFC3339, v)
		}
	}
	return r, nil
}

// Alive reports whether pid names a live process, via a signal-0 probe (send
// no signal, just test existence/permission). EPERM means it exists but is
// owned elsewhere — still alive for our purposes.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// GroupAlive reports whether ANY member of the process group still exists.
// This is the runaway check: the leader can exit while forked children keep
// running, and the group lives as long as one member does.
func GroupAlive(pgid int) bool {
	if pgid <= 0 {
		return false
	}
	err := syscall.Kill(-pgid, 0)
	return err == nil || err == syscall.EPERM
}

// Usage is a resource snapshot of one process group.
type Usage struct {
	Procs  int     // members of the group currently alive
	RSSKB  int     // summed resident memory (KB)
	CPUPct float64 // summed %CPU across the group
	GPUMiB int     // summed GPU memory (nvidia only); -1 when unmeasurable
}

// SampleGroup sums RSS and %CPU over every process whose group id is pgid, by
// parsing a single `ps` snapshot (BSD/GNU compatible keywords). GPU is layered
// on best-effort; on a machine with no nvidia-smi it stays -1 (reported as
// n/a, never faked).
func SampleGroup(pgid int) Usage {
	u := Usage{GPUMiB: -1}
	if pgid <= 0 {
		return u
	}
	out, err := exec.Command("ps", "-A", "-o", "pgid=,pid=,rss=,%cpu=").Output()
	if err != nil {
		return u
	}
	var pids []int
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 {
			continue
		}
		pg, _ := strconv.Atoi(fields[0])
		if pg != pgid {
			continue
		}
		pid, _ := strconv.Atoi(fields[1])
		rss, _ := strconv.Atoi(fields[2])
		cpu, _ := strconv.ParseFloat(fields[3], 64)
		u.Procs++
		u.RSSKB += rss
		u.CPUPct += cpu
		pids = append(pids, pid)
	}
	if g, ok := gpuByPID(pids); ok {
		u.GPUMiB = g
	}
	return u
}

// gpuByPID sums GPU memory held by any of pids via nvidia-smi's compute-app
// table. Returns ok=false when nvidia-smi is absent (Apple silicon, CPU box)
// or reports nothing for these pids — the caller then shows n/a.
func gpuByPID(pids []int) (int, bool) {
	if len(pids) == 0 {
		return 0, false
	}
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return 0, false
	}
	out, err := exec.Command("nvidia-smi",
		"--query-compute-apps=pid,used_memory", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return 0, false
	}
	want := make(map[int]bool, len(pids))
	for _, p := range pids {
		want[p] = true
	}
	total, matched := 0, false
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		parts := strings.Split(sc.Text(), ",")
		if len(parts) < 2 {
			continue
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		if !want[pid] {
			continue
		}
		mib, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		total += mib
		matched = true
	}
	return total, matched
}

// KillTree terminates a whole process group: SIGTERM first, so the tree gets a
// chance to flush and exit cleanly, then SIGKILL after grace if anything
// survives. Signalling the NEGATIVE pgid reaches EVERY member — the agent AND
// every subprocess it forked — which is the entire point: no orphaned runaways
// left holding RAM/CPU/GPU. termed reports the SIGTERM landed; killed reports a
// SIGKILL was needed.
func KillTree(pgid int, grace time.Duration) (termed, killed bool) {
	if pgid <= 0 {
		return false, false
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err == nil {
		termed = true
	}
	for waited := time.Duration(0); waited < grace; waited += 100 * time.Millisecond {
		if !GroupAlive(pgid) {
			return termed, false
		}
		time.Sleep(100 * time.Millisecond)
	}
	if GroupAlive(pgid) {
		if err := syscall.Kill(-pgid, syscall.SIGKILL); err == nil {
			killed = true
		}
	}
	return termed, killed
}
