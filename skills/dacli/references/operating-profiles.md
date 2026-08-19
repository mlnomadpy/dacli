# Operating profiles

Use inspect for diagnosis; a single task for one bounded edit; a supervised wave for independent ready tasks; and `loop` only for repeatable backlog work. For a wave, use `next --parallel <width>`, worktrees with disjoint claims, `wait`, `sync`, verification, and the project's landing policy. Set WIP no higher than the number of safe claims, reviewers, and landing capacity.

For a loop, always preview first: `dacli loop --project <project> --width N --max-cycles N --dry-run`. The real run keeps a finite cycle count, per-cycle `--max-tokens`, rolling `--window-tokens`/`--token-window`, `--halt-after-idle`, and an explicit `--idle` backoff. `--yolo` is a deliberate override, never a synonym for unlimited spend or release publication.
