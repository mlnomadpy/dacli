# Critical path and GitHub landing

List existing open and active tasks, preview `github pull <project> --dry-run`, and compare the proposed issue titles before adoption: pull prevents duplicate issue mappings but does not perform semantic deduplication. After `github pull <project>`, estimate with `task estimate <ref> --estimate o,m,p`, then use `critical-path --project <project>` and `next --project <project> --parallel <width>`. Critical-path output is a schedule aid, not permission to violate WIP, claims, or review gates.

Create a locally planned dependent task with `task add "<title>" --project <project> --depends-on <ref[:TYPE]> --accept "<observable criterion>"`. The shipped CLI cannot add a dependency edge to an already-adopted issue task; stop and reconcile that backlog rather than guessing or scheduling a false graph. After a verified task returns, use this minimum handoff:

```bash
dacli sync
dacli task check <ref> --n <n> --verify "<command>"
dacli push <ref>
dacli pr --task <ref> --with-verdicts
dacli pr status --task <ref>
dacli accept <ref> --verify "<command>"
dacli retro <ref> --well "..." --bad "..." --improve "..."
```

Keep GitHub as the visible projection: preview outbound changes with `github push <project> --dry-run`; push task branches with `push <ref>`, then open them with `pr --task <ref> --with-verdicts`; verify checks and `pr status` before closing/syncing. Add `--auto` only when the repository's required checks and review policy make auto-merge trustworthy. A merge, CI result, or API call that cannot be observed is unverified. Never create tags or releases as part of ordinary loop work.
