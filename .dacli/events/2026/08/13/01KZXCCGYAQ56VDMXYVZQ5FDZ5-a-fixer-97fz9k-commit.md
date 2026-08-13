---
id: 01KZXCCGYAQ56VDMXYVZQ5FDZ5
kind: event
event_kind: commit
created: 2026-08-13T10:58:45Z
created_by: a-fixer-97fz9k
about: "[[t-01KZX7PXQBEVM1M0N2BKWYD4RK]]"
origin: agent
applied: true
---
99d531b 406: wire provider limits into runtime execution

Classify failed spawn and supervise turns, persist fallbackable cooldowns,
and refuse or follow only role-declared fallback chains before launch. Persist
fallback_to so the explicit policy survives role reloads.

Red-test mutations:
- disabling failure recording: TestSpawnClassifiesProviderFailureAndRecordsOnlyFallbackableCooldown/rate-limit failed with breaker open = false, want true
- removing the fallbackability guard: TestResolveLaunchPermanentAndPolicyCooldownsNeverFallback failed with exit 0, want policy pause
role: fixer
