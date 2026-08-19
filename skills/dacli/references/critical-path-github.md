# Critical path and GitHub landing

Pull human work, deduplicate, estimate it, and add dependency edges before scheduling: `github pull <project>`, `task list --status open --project <project>`, `task estimate <ref> --estimate o,m,p`, `critical-path --project <project>`, and `next --project <project> --parallel N`. Critical-path output is a schedule aid, not permission to violate WIP, claims, or review gates.

Keep GitHub as the visible projection: preview outbound changes with `github push <project> --dry-run`; open task branches with `push --task <ref>` and `pr --task <ref> --with-verdicts --auto`; verify checks and `pr status` before closing/syncing. A merge, CI result, or API call that cannot be observed is unverified. Never create tags or releases as part of ordinary loop work.
