package execution

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// RunGuardian is the private process-group leader used by every runtime
// launch. It deliberately does not exec the runtime: retaining this process's
// PID/start identity gives later wait/kill invocations durable proof that the
// numeric group still belongs to the recorded run (tasks 177 and 369).
func RunGuardian(argv []string) int {
	exitFile := ""
	if len(argv) >= 2 && argv[0] == "--exit-file" {
		exitFile, argv = argv[1], argv[2:]
	}
	finish := func(code int) int {
		if exitFile != "" {
			_ = os.WriteFile(exitFile, []byte(fmt.Sprintf("%d\n", code)), 0o644)
		}
		return code
	}
	if len(argv) == 0 {
		return finish(2)
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return finish(1)
	}
	err := cmd.Wait()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			code = 1
		} else {
			code = exitErr.ExitCode()
		}
	}
	for procmonGroupHasOtherLiveProcess(os.Getpid()) {
		time.Sleep(25 * time.Millisecond)
	}
	return finish(code)
}
