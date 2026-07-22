---
id: 01KY5GXKQH8YWABK3GPN1CRDY4
kind: event
event_kind: finding
created: 2026-07-22T18:20:31Z
created_by: a-1zhjz6t2va
about: [[t-01KY5GP5S0V2MFQC2R2EFVEA5A]]
origin: agent
applied: true
---
estimate ignores the token-per-point unit that calibrate calls PREFERRED

internal/features/insight/insight.go:388-404: cmdEstimate's empirical-band inversion filters cal.Samples by band and projects med*Te using ONLY s.Ratio() (wall-clock claim->completion). It never surfaces TokenRatio(), even when the band has token samples. But cmdCalibrate (same file, :725,:734,:748-749) labels the tokens/point band 'PREFERRED (real unit; wall-clock above is the fallback)' and 'tokens ARE the estimate'. So 'dacli estimate' contradicts 'dacli calibrate': the command that produces THE estimate for a task uses the unit calibrate explicitly demotes to the fallback, and a runtime that reports usage gets no token-based estimate. When a band HasTokens, estimate should lead with med(TokenRatio)*Te and mark wall-clock the fallback, mirroring calibrate.
