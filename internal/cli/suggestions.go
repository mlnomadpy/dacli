package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
)

// guidedUsageError retains the ordinary exit-2 usage error while exposing
// structured, registry-derived recovery to CLI and MCP callers.
type guidedUsageError struct {
	err         error
	suggestions []string
	nextActions []string
}

func (e guidedUsageError) Error() string              { return e.err.Error() }
func (e guidedUsageError) Unwrap() error              { return e.err }
func (e guidedUsageError) ErrorSuggestions() []string { return e.suggestions }
func (e guidedUsageError) ErrorNextActions() []string { return e.nextActions }

type commandGuidance struct {
	suggestions []string
	nextActions []string
	family      string
}

// guidanceCommandTable breaks the aggregate table's static initializer cycle:
// the table contains mcp serve, whose executor also uses this guidance engine.
var guidanceCommandTable func() []Command

func registeredCommands() []Command {
	if guidanceCommandTable == nil {
		return nil
	}
	return guidanceCommandTable()
}

func unknownCommandError(args []string) error {
	g := commandGuidanceFor(args)
	return guidedUsageError{
		err:         clikit.Usagef("unknown command %q", strings.Join(args, " ")),
		suggestions: g.suggestions,
		nextActions: g.nextActions,
	}
}

func commandGuidanceFor(args []string) commandGuidance {
	if len(args) == 0 {
		return commandGuidance{nextActions: []string{"dacli help"}}
	}

	// A bare `show` has two plausible, read-only meanings. Never guess the
	// object type and never include a mutating command in this ambiguity set.
	if editDistance(strings.ToLower(args[0]), "show") <= 1 && len(args) >= 2 {
		tail := strings.Join(args[1:], " ")
		return commandGuidance{
			suggestions: []string{"dacli project show " + tail, "dacli task show " + tail},
			nextActions: []string{"choose the object family explicitly; no suggestion was executed"},
		}
	}

	family := args[0]
	familyCommands := commandsUnderFamily(family)
	if len(familyCommands) > 0 {
		if len(args) >= 2 {
			if suggested := closestFamilyCommand(familyCommands, args[1]); suggested != nil {
				return commandGuidance{
					suggestions: []string{renderSuggestion(*suggested, args[2:])},
					nextActions: []string{"review the suggested command; it was not executed"},
					family:      family,
				}
			}
			if show := commandByPath(family + " show"); show != nil && looksLikeReference(args[1]) {
				return commandGuidance{
					suggestions: []string{renderSuggestion(*show, args[1:])},
					nextActions: []string{"review the suggested read-only command; it was not executed"},
					family:      family,
				}
			}
		}
		return commandGuidance{
			nextActions: []string{fmt.Sprintf("dacli %s --help", family)},
			family:      family,
		}
	}

	if suggested := closestCommand(args); suggested != nil {
		return commandGuidance{
			suggestions: []string{renderClosestSuggestion(*suggested, args)},
			nextActions: []string{"review the suggested command; it was not executed"},
		}
	}
	return commandGuidance{nextActions: []string{"dacli help", "dacli help --all"}}
}

func commandsUnderFamily(family string) []Command {
	prefix := family + " "
	var out []Command
	for _, cmd := range registeredCommands() {
		if strings.HasPrefix(cmd.Path, prefix) {
			out = append(out, cmd)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func commandByPath(path string) *Command {
	registered := registeredCommands()
	for i := range registered {
		if registered[i].Path == path {
			return &registered[i]
		}
	}
	return nil
}

func closestFamilyCommand(candidates []Command, leaf string) *Command {
	type ranked struct {
		cmd      Command
		distance int
	}
	var matches []ranked
	for _, candidate := range candidates {
		parts := strings.Fields(candidate.Path)
		if len(parts) < 2 {
			continue
		}
		distance := editDistance(strings.ToLower(leaf), strings.ToLower(parts[1]))
		if distance <= typoThreshold(leaf) {
			matches = append(matches, ranked{candidate, distance})
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].distance != matches[j].distance {
			return matches[i].distance < matches[j].distance
		}
		return matches[i].cmd.Path < matches[j].cmd.Path
	})
	return &matches[0].cmd
}

func closestCommand(args []string) *Command {
	input := strings.ToLower(args[0])
	type ranked struct {
		cmd      Command
		distance int
	}
	var matches []ranked
	for _, candidate := range registeredCommands() {
		parts := strings.Fields(candidate.Path)
		first := parts[0]
		distance := editDistance(input, strings.ToLower(first))
		if len(args) >= 2 && len(parts) >= 2 {
			distance += editDistance(strings.ToLower(args[1]), strings.ToLower(parts[1]))
		}
		if distance <= typoThreshold(input) {
			matches = append(matches, ranked{candidate, distance})
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].distance != matches[j].distance {
			return matches[i].distance < matches[j].distance
		}
		// Prefer read-only candidates when only the family is known. This keeps
		// ambiguous correction from steering callers toward a mutation.
		if matches[i].cmd.Mutates != matches[j].cmd.Mutates {
			return !matches[i].cmd.Mutates
		}
		return matches[i].cmd.Path < matches[j].cmd.Path
	})
	return &matches[0].cmd
}

func renderSuggestion(cmd Command, rest []string) string {
	parts := []string{"dacli", cmd.Path}
	parts = append(parts, rest...)
	return strings.Join(parts, " ")
}

func renderClosestSuggestion(cmd Command, args []string) string {
	consumed := min(len(strings.Fields(cmd.Path)), len(args))
	return renderSuggestion(cmd, args[consumed:])
}

func looksLikeReference(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-")
}

func typoThreshold(value string) int {
	if len(value) <= 3 {
		return 1
	}
	return 2
}

func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	previous := make([]int, len(br)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, ra := range ar {
		current := make([]int, len(br)+1)
		current[0] = i + 1
		for j, rb := range br {
			cost := 0
			if ra != rb {
				cost = 1
			}
			current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(br)]
}
