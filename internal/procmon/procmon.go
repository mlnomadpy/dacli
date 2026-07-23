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
	// PIDStart is the OS-reported start time of PID captured at spawn (ps
	// lstart). A PID is recycled by the kernel once the original process exits,
	// so PID alone cannot prove a proc.txt still names OUR agent. Re-reading the
	// live PID's start time and comparing it to PIDStart rejects a recycled PID:
	// an unrelated process that inherited the number started at a different time.
	// Empty on legacy records (best-effort: fall back to a bare liveness probe).
	PIDStart string
	Started  time.Time
	Claims   []string // repo paths this agent declared it will edit (advisory lock)
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
	if r.PIDStart != "" {
		fmt.Fprintf(&b, "pid_start: %s\n", r.PIDStart)
	}
	fmt.Fprintf(&b, "started: %s\n", r.Started.UTC().Format(time.RFC3339))
	if len(r.Claims) > 0 {
		fmt.Fprintf(&b, "claims: %s\n", strings.Join(r.Claims, ","))
	}
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
		case "pid_start":
			r.PIDStart = v
		case "started":
			r.Started, _ = time.Parse(time.RFC3339, v)
		case "claims":
			for _, p := range strings.Split(v, ",") {
				if p = strings.TrimSpace(p); p != "" {
					r.Claims = append(r.Claims, p)
				}
			}
		}
	}
	return r, nil
}

// PathsOverlap reports whether any path in a claims the same tree as any path
// in b — i.e. one is the other, or a path-segment prefix of the other
// (internal/store vs internal/store/roles.go overlap; internal/store vs
// internal/storefront do NOT). Used to refuse two live agents editing the same
// files in parallel.
func PathsOverlap(a, b []string) (string, string, bool) {
	clean := func(p string) string { return strings.Trim(strings.TrimSpace(p), "/") }
	prefix := func(p, q string) bool {
		p, q = clean(p), clean(q)
		return p == q || strings.HasPrefix(q, p+"/")
	}
	for _, x := range a {
		for _, y := range b {
			if prefix(x, y) || prefix(y, x) {
				return x, y, true
			}
		}
	}
	return "", "", false
}

// ProcStart returns pid's OS start time as reported by `ps -o lstart=` (an
// absolute wall-clock stamp, stable for the life of the process and identical
// on macOS and Linux). ok=false when the process is gone or ps cannot read it.
// This is the identity fingerprint used to detect a recycled PID.
func ProcStart(pid int) (string, bool) {
	if pid <= 0 {
		return "", false
	}
	out, err := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", false
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", false
	}
	return s, true
}

// AliveIdentity reports whether pid is alive AND is still the same process that
// was recorded with start time wantStart. A live PID whose start time no longer
// matches wantStart was recycled by the kernel onto an unrelated process, so it
// is NOT our agent — treat it as gone. This is what stops a stale proc.txt from
// resurfacing a dead run as live or steering KillTree onto someone else's group.
// A record with no recorded start time (wantStart == "", e.g. a legacy proc.txt)
// falls back to a bare liveness probe, preserving prior behavior.
func AliveIdentity(pid int, wantStart string) bool {
	if !Alive(pid) {
		return false
	}
	if wantStart == "" {
		return true
	}
	got, ok := ProcStart(pid)
	if !ok {
		// The PID is live (signal-0 succeeded) but we cannot read its start time
		// to confirm identity. Refuse to vouch for it rather than risk sampling
		// or killing a recycled PID.
		return false
	}
	return got == wantStart
}

// AliveRecord is AliveIdentity applied to a run's Record — the identity-checked
// liveness test every reader (agents/kill/wait/logs) should use in place of a
// bare Alive(rec.PID).
func AliveRecord(r Record) bool { return AliveIdentity(r.PID, r.PIDStart) }

// Usage is a resource snapshot of one process group.
type Usage struct {
	Procs  int     // members of the group currently alive
	RSSKB  int     // summed resident memory (KB)
	CPUPct float64 // summed ps %cpu — a per-process LIFETIME AVERAGE, not current load
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
