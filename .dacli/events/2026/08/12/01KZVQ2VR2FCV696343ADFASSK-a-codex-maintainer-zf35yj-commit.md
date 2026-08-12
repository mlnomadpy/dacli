---
id: 01KZVQ2VR2FCV696343ADFASSK
kind: event
event_kind: commit
created: 2026-08-12T19:27:14Z
created_by: a-codex-maintainer-zf35yj
about: "[[t-01KZV16EPCRG5DXDSVW08TSSH6]]"
origin: agent
applied: true
---
dc9feec 368: make markdown store writes power-loss durable

Sync record data before rename and the containing directory after it, and route runtime probe cache writes through the same primitive.

Red test: TestWriteBytesSyncsDataBeforeRenameAndDirectoryAfter failed: durability operation order = mkdir,create temp,write,chmod,close file,rename,open dir,close dir; missing sync file and sync dir.
role: codex-maintainer
