// Package buildinfo holds build-time identity: the version and the tool's
// own repo (where `dacli report` files bugs). Version is overridable at
// build with -ldflags "-X .../buildinfo.Version=v1.2.3".
package buildinfo

// Version is the dacli release. "dev" until a build stamps it.
var Version = "dev"

// Repo is the tool's own GitHub repository — the default target for
// `dacli report`, so agent-hit bugs flow back to the tool, not the user's
// project. Overridable per invocation (--repo) or env (DACLI_REPORT_REPO).
const Repo = "mlnomadpy/dacli"
