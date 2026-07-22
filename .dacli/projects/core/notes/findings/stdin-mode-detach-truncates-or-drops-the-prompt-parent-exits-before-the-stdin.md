---
id: f-stdin-mode-detach-truncates-or-drops-the-prompt-parent-exits-before-the-stdin
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-1s5dxes66y
about: [[t-01KY59FNEAAYDZ0PCTKE0HCBVA]]
source_event: 01KY59NYECGWADBYXKX8T9FYZR
---
# stdin-mode --detach truncates or drops the prompt: parent exits before the stdin-copy goroutine finishes
execution.go execRuntime detach path (816-841): for rt.Mode=='stdin' it sets cmd.Stdin=strings.NewReader(prompt) (827-829), calls cmd.Start() (830), onStart, then cmd.Process.Release() (840) and returns — it NEVER calls cmd.Wait(). In os/exec, a non-*os.File Stdin makes Start() create an os.Pipe and spawn a background goroutine that copies prompt->pipe; that copy is normally drained by Wait(). Release() does not wait for it. When the real 'dacli spawn --detach' process returns from cmdSpawn and exits, that goroutine dies and the pipe's write end closes, so the child reads EOF early. A prompt larger than the pipe buffer (~64KB; briefs here are ~15k tokens and can exceed it) is truncated or lost entirely. The generic-exec preset (execution.go:71) is stdin-mode, so 'dacli spawn --detach --runtime generic-exec' is the concrete broken case; arg-mode runtimes (claude-code, :59) pass the prompt as argv and are unaffected. Fix: for stdin+detach, write the prompt to a temp file and set cmd.Stdin to that *os.File (fd inherited, no parent goroutine), or refuse stdin+detach.
