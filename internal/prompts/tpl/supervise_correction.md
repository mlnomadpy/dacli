
**SYSTEM:** supervisor: turn {{.Turn}} of {{.MaxTurns}}. The work is not done: these acceptance criteria are still unmet.
{{- range .Unmet}}
- {{.}}
{{- end}}
Fix exactly these and report each through dacli; everything else already passed, so do not touch it. If a criterion is unmet because it is blocked (not because you skipped it), say so with `dacli ask` rather than looping.
