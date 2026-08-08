---
id: f-createtask-s-seq-scan-then-write-raced-across-processes
kind: note
note_kind: finding
created: 2026-08-04T00:09:59Z
created_by: a-fixer-q3xzg4
about: "[[209]]"
severity: major
---
# CreateTask's seq scan-then-write raced across processes
internal/store/store.go CreateTask computed seq by scanning ListTasks then wrote the task file via mdstore.WriteFile, which os.Rename's unconditionally (no O_EXCL) -- two concurrent CreateTask calls (e.g. two agents filing at once) could compute the same next seq before either file landed. Repro'd with a 20-goroutine concurrent CreateTask test (internal/store/task_seq_test.go TestCreateTaskConcurrentGetsDistinctSeqs): reliably produced duplicate seqs (e.g. two files both claiming seq 1) across every one of 5 runs before the fix. Fixed by acquireSeqLock (store.go), an O_EXCL marker-file lock per project that serializes the scan+write; verified green under -race x10 plus the full test suite (go test -exec 'env -u DACLI_AGENT' ./...).
