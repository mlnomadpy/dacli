# Critical path and GitHub landing

Pull human work, deduplicate, estimate it, and add dependency edges before scheduling: `github pull <project>`, `task list --status open --project <project>`, `task estimate <ref> --estimate o,m,p`, `critical-path --project <project>`, and `next --project <project> --parallel <width>`. Critical-path output is a schedule aid, not permission to violate WIP, claims, or review gates.

Create a locally planned dependent task with `task add "<title>" --project <project> --depends-on <ref[:TYPE]> --accept "<observable criterion>"`; issue adoption does not make a guessed dependency safe. After a verified task returns, use this minimum handoff:

```bash
dacli sync
dacli task check <ref> --n <n> --verify "<command>"
dacli github push <project> <ref>
dacli pr --task <ref> --with-verdicts --auto
dacli pr status --task <ref>
dacli accept <ref> --verify "<command>"
dacli retro <ref> --well "..." --bad "..." --improve "..."
```

Keep GitHub as the visible projection: preview outbound changes with `github push <project> --dry-run`; open task branches with `github push <project> <ref>` and `pr --task <ref> --with-verdicts --auto`; verify checks and `pr status` before closing/syncing. A merge, CI result, or API call that cannot be observed is unverified. Never create tags or releases as part of ordinary loop work.
