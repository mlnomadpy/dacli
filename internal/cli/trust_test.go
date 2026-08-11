package cli

import (
	"os"
	"strings"
	"testing"
)

// docs/TRUST.md (task 353) is the one place the trust/taint model, the
// untrusted-content boundary, secret handling, and the undo path for every
// `Mutates` command are stated together, instead of inferred from scattered
// code comments. A doc that claims to enumerate a live command table drifts
// the moment a new mutating command ships and nobody updates it by hand — the
// exact failure mode issue #436 named for flag documentation before
// TestFlagTakingCommandsDocumentTheirFlags existed to catch it.
//
// This is that same guard for TRUST.md: every command carrying
// `Mutates: true` in the live table must have its Path named somewhere in the
// doc. Enumerated from `commands`, not sampled or hand-copied, so the set
// this test checks can never itself drift from the set the dispatcher
// actually gates.
func TestTrustDocListsEveryMutatingCommand(t *testing.T) {
	raw, err := os.ReadFile("../../docs/TRUST.md")
	if err != nil {
		t.Fatalf("docs/TRUST.md: %v", err)
	}
	text := string(raw)

	var checked int
	var missing []string
	for i := range commands {
		c := &commands[i]
		if !c.Mutates {
			continue
		}
		checked++
		// A command path appears in the doc as a backtick-quoted token, e.g.
		// `github push` or `task rm` — matching the bare path (not
		// backtick-wrapped) is enough and avoids the table needing to
		// reproduce the doc's exact Markdown escaping.
		if !strings.Contains(text, c.Path) {
			missing = append(missing, c.Path)
		}
	}
	if checked == 0 {
		t.Fatal("no command declares Mutates: true — this test measured nothing")
	}
	if len(missing) > 0 {
		t.Errorf("docs/TRUST.md does not name %d of %d Mutates commands — the trust doc has drifted from the live command table:\n  %s",
			len(missing), checked, strings.Join(missing, "\n  "))
	}
}
