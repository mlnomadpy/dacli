# Critical path and GitHub landing

List existing open and active tasks, inspect open GitHub issues, and preview inbound adoption with `github pull <project> --dry-run` (`github sync <project> --dry-run` previews both directions); then compare the proposed issue titles before adoption. Pull prevents duplicate issue mappings but does not perform semantic deduplication. Before pull, make each issue's `## Acceptance criteria` checkbox list independently checkable; the shipped CLI has no task-edit command that can add missing criteria to an adopted task. After `github pull <project>`, estimate with `task estimate <ref> --estimate o,m,p`.

Create a locally planned dependent task with `task add "<title>" --project <project> --depends-on <ref[:TYPE]> --accept "<observable criterion>"`. The shipped CLI cannot add a dependency edge to an already-adopted issue task. If an adopted graph is incomplete, preserve its GitHub mapping and history, record it with `note add finding "Missing adopted-task dependency edges" --project <project> --about <ref> --body "<missing edges and source>"`, and stop the dependency-sensitive wave; do not delete and recreate tasks. Until a dependency-edit command ships, work only explicit task refs whose independence you verified. Do not use `critical-path --project <project>` or `next --project <project> --parallel <width>` as authoritative scheduling until the recorded graph is truthful. Critical-path output is a schedule aid, not permission to violate WIP, claims, or review gates. After a verified task returns, use this minimum handoff:

```bash
dacli sync
dacli task check <ref> --n <n> --verify "<command>"
dacli push <ref>
dacli pr --task <ref> --with-verdicts
dacli pr status --task <ref>
dacli accept <ref> --verify "<command>"
dacli retro <task-or-project-ref> --well "..." --bad "..." --improve "..."
```

Keep GitHub as the visible projection: preview outbound changes with `github push <project> --dry-run`; push task branches with `push <ref>`, then open them with `pr --task <ref> --with-verdicts`. Before owner acceptance or issue closure, observe both the merged PR through `pr status` and its commit on freshly inspected trunk. `ship` is the separate wave transaction that owns accept-plus-integrate for its reviewed task window. Add `--auto` only when the repository's required checks and review policy make auto-merge trustworthy. A merge, CI result, or API call that cannot be observed is unverified. Never create tags or releases as part of ordinary loop work.
