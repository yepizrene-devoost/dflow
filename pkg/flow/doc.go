// Package flow is dflow's pure branching domain.
//
// It owns the branching decisions themselves — the `.dflow.yaml` schema and its
// backward-compatible YAML rules, the branch-type model, and finish planning —
// and nothing else: no I/O, no printing, and no filesystem or network access.
//
// Because it is pure, any client can resolve the same decisions from the same
// configuration: the CLI, a future TUI, a forge provider or a deploy provider.
package flow
