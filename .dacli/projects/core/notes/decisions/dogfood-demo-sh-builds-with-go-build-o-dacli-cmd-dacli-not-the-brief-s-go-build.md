---
id: d-dogfood-demo-sh-builds-with-go-build-o-dacli-cmd-dacli-not-the-brief-s-go-build
kind: note
note_kind: decision
created: 2026-07-22T13:57:55Z
created_by: a-2xx9dxvf26
about: [[030]]
---
# dogfood-demo.sh builds with 'go build -o ./dacli ./cmd/dacli', not the brief's 'go build -o ./dacli .'
## Chose
dogfood-demo.sh builds with 'go build -o ./dacli ./cmd/dacli', not the brief's 'go build -o ./dacli .'
## Rejected
the brief's 'go build -o ./dacli .' at repo root
## Because
main package lives at cmd/dacli/main.go; 'go build .' at repo root fails with 'no Go files'. The demo must actually build, so the script targets ./cmd/dacli.
