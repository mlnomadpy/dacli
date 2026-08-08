---
id: d-151-hand-authored-the-shadcn-vue-foundation-single-root-dark-theme-instead-of
kind: note
note_kind: decision
created: 2026-07-26T17:16:20Z
created_by: a-12vbv8bcwh
about: [[151]]
---
# 151: hand-authored the shadcn-vue foundation (single :root dark theme) instead of running the shadcn-vue init CLI
## Chose
151: hand-authored the shadcn-vue foundation (single :root dark theme) instead of running the shadcn-vue init CLI
## Rejected
npx shadcn-vue@latest init + add, and a .dark-class-gated theme
## Because
the init/add CLI is interactive and network-fetches the registry — unusable headless; authoring the same artifacts (components.json, @tailwindcss/vite, src/assets/index.css tokens, src/lib/utils cn, src/components/ui/{button,card,badge,table,progress,separator,tooltip}) by hand is deterministic and reviewable. The console has no light mode (DESIGN.md), so the shadcn tokens map 1:1 onto the mission-control palette (#0f1115 bg / #161a22 panel / #4f8cff blue primary / #e5534b destructive / #3fb950 success) directly on :root rather than under a .dark class that nothing toggles. tokens.css is kept (unlayered, so it still wins over Tailwind base) so the not-yet-refactored sections render unchanged.
