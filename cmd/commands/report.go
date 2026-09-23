package commands

import "github.com/yepizrene-devoost/dflow/pkg/flow"

// This file holds the machine-readable report shapes that more than one command
// emits. They live here, not in the command that happened to declare them
// first, so the contract a script parses is one shared definition instead of a
// copy per command.

// targetView is the machine-readable shape of one finish target: the branch and
// the merge mode that will actually apply to it.
//
// `dflow status --json` and `dflow finish --dry-run --json` both report it, so a
// script sees one shape for the same concept everywhere.
type targetView struct {
	Branch    string `json:"branch"`
	MergeMode string `json:"merge_mode"`
}

// targetViews converts a resolved plan's targets into their reported shape.
//
// It always returns a non-nil slice so an empty plan marshals as `[]` rather
// than `null`: "no targets" is an answer, not a missing value.
func targetViews(targets []flow.FinishTarget) []targetView {
	views := make([]targetView, 0, len(targets))
	for _, target := range targets {
		views = append(views, targetView{Branch: target.Branch, MergeMode: target.MergeMode})
	}
	return views
}
