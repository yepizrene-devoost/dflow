// Package agent renders the agent-facing workflow document for a dflow project.
//
// The generated markdown is derived entirely from the parsed `.dflow.yaml`, so
// the rules an agent reads and the rules dflow enforces cannot drift apart:
// changing the configuration changes both.
package agent

import (
	"bytes"
	"sort"
	"strings"
	"text/template"

	"github.com/yepizrene-devoost/dflow/pkg/flow"
)

// branchTypeRow is one row of the Branch Types table.
type branchTypeRow struct {
	Type          string
	Aliases       string
	Prefix        string
	Base          string
	FinishTargets string
}

// mergeRuleRow is one row of the Merge Rules table.
type mergeRuleRow struct {
	Branch   string
	Mode     string
	Behavior string
}

// templateData is the view model handed to the workflow template.
//
// The fields must stay exported: text/template resolves only exported struct
// fields, and the template addresses them by name.
type templateData struct {
	BranchTypes      []branchTypeRow
	MergeRules       []mergeRuleRow
	DefaultMergeMode string
	DefaultBranch    string
}

// markdownBacktick stands in for a backtick inside the raw template source.
//
// Go raw string literals cannot contain a backtick, and the workflow document is
// full of markdown code spans, so the template is written with this
// near-identical glyph and the real delimiter is substituted once, before
// parsing. That keeps the template readable as the verbatim document it renders.
const markdownBacktick = "｀"

// templateBacktick is the character markdownBacktick stands in for.
const templateBacktick = "`"

// mergeModeAuto and mergeModeManual are the only merge modes dflow accepts.
const (
	mergeModeAuto   = "auto"
	mergeModeManual = "manual"
)

// agentTemplate renders the markdown workflow document.
var agentTemplate = template.Must(template.New("agent").Parse(
	strings.ReplaceAll(agentTemplateSource, markdownBacktick, templateBacktick),
))

// GenerateAgentDoc renders the agent workflow markdown for cfg.
//
// Every dynamic value — branch prefixes, bases, finish targets and merge rules —
// comes from cfg, so the returned document always describes the configuration
// that produced it. The template is a package constant with fixed actions, so a
// rendering failure here would be a programmer error rather than a configuration
// error; it is reported inline instead of panicking so the caller's write path
// stays infallible.
func GenerateAgentDoc(cfg *flow.Config) []byte {
	data := templateData{
		BranchTypes:      branchTypeRows(cfg),
		MergeRules:       mergeRuleRows(cfg),
		DefaultMergeMode: cfg.Workflow.DefaultMergeMode,
		DefaultBranch:    cfg.Branches.Develop,
	}

	var out bytes.Buffer
	if err := agentTemplate.Execute(&out, data); err != nil {
		return []byte("# dflow Workflow\n\n<!-- failed to render the agent workflow: " + err.Error() + " -->\n")
	}
	return out.Bytes()
}

// branchTypeRows builds the Branch Types table in its fixed display order.
func branchTypeRows(cfg *flow.Config) []branchTypeRow {
	return []branchTypeRow{
		{
			Type:          string(flow.BranchTypeFeature),
			Aliases:       "feat",
			Prefix:        cfg.Branches.Features,
			Base:          cfg.Flow.Feature.Base,
			FinishTargets: strings.Join(cfg.Flow.Feature.FinishTargets, ", "),
		},
		{
			Type:          string(flow.BranchTypeRelease),
			Aliases:       "—",
			Prefix:        cfg.Branches.Releases,
			Base:          cfg.Flow.Release.Base,
			FinishTargets: strings.Join(cfg.Flow.Release.FinishTargets, ", "),
		},
		{
			Type:          string(flow.BranchTypeHotfix),
			Aliases:       "hot, fix",
			Prefix:        cfg.Branches.Hotfixes,
			Base:          cfg.Flow.Hotfix.Base,
			FinishTargets: strings.Join(cfg.Flow.Hotfix.FinishTargets, ", "),
		},
		{
			Type:          string(flow.BranchTypeBugfix),
			Aliases:       "bug",
			Prefix:        cfg.Branches.Bugfixes,
			Base:          cfg.Flow.Bugfix.Base,
			FinishTargets: strings.Join(cfg.Flow.Bugfix.FinishTargets, ", "),
		},
	}
}

// mergeRuleRows builds the Merge Rules table.
//
// The branch names are sorted so the generated document is byte-for-byte stable
// across runs: ranging a Go map directly would reorder the rows arbitrarily and
// make the output non-reproducible.
func mergeRuleRows(cfg *flow.Config) []mergeRuleRow {
	branches := make([]string, 0, len(cfg.Workflow.BranchRules))
	for branch := range cfg.Workflow.BranchRules {
		branches = append(branches, branch)
	}
	sort.Strings(branches)

	rows := make([]mergeRuleRow, 0, len(branches))
	for _, branch := range branches {
		mode := cfg.Workflow.BranchRules[branch].MergeMode
		rows = append(rows, mergeRuleRow{
			Branch:   branch,
			Mode:     mode,
			Behavior: mergeBehavior(mode),
		})
	}
	return rows
}

// mergeBehavior spells out what a merge mode means for an agent acting on the
// repository. An unrecognised mode is named as such instead of being silently
// rendered as an empty cell.
func mergeBehavior(mode string) string {
	switch mode {
	case mergeModeAuto:
		return "Direct merge via `dflow finish`"
	case mergeModeManual:
		return "Open a Pull Request"
	default:
		return "Unknown merge mode"
	}
}

// agentTemplateSource is the workflow document template.
//
// It is a Go raw string; every backtick in the rendered markdown is written here
// as markdownBacktick and substituted before parsing.
const agentTemplateSource = `---
name: dflow
description: "Trigger: dflow start, dflow finish, branch workflow, merge rules, finish targets, dflow status, dflow delete. Project's dflow configuration and branch workflow."
metadata:
  source: .dflow.yaml
  generated_by: dflow agent
---

# dflow Workflow

<!-- Generated from .dflow.yaml — regenerate with: dflow agent -->

## Branch Types

| Type | Aliases | Prefix | Base | Finish Targets |
|---|---|---|---|---|
{{range .BranchTypes}}| {{.Type}} | {{.Aliases}} | {{.Prefix}} | {{.Base}} | {{.FinishTargets}} |
{{end}}
## Merge Rules

| Branch | Mode | Behavior |
|---|---|---|
{{range .MergeRules}}| {{.Branch}} | {{.Mode}} | {{.Behavior}} |
{{end}}
Default mode: {{.DefaultMergeMode}}

## Commands

| Command | Purpose |
| --- | --- |
| ｀dflow init｀ | Initialize ｀.dflow.yaml｀ (run once per project; refuses to overwrite an existing file, use ｀--force｀ to regenerate it) |
| ｀dflow agent｀ | Generate an agent workflow file from ｀.dflow.yaml｀ |
| ｀dflow start <type> <name>｀ | Create and switch to a work branch |
| ｀dflow finish｀ | Merge the current branch into its configured ｀auto｀ targets |
| ｀dflow finish --dry-run｀ | Preview the finish plan without merging or pushing |
| ｀dflow finish --dry-run --json｀ | Preview the finish plan as a single JSON document |
| ｀dflow finish --delete｀ | Also delete the branch when no ｀manual｀ targets remain |
| ｀dflow finish --no-push｀ | Merge without publishing the work branch |
| ｀dflow status [--json]｀ | Report branch, detected type, resolved targets and Git state |
| ｀dflow delete <branch> [--yes]｀ | Delete a branch locally and remotely; idempotent |
| ｀dflow update [--check] [--force] [--yes] [--json]｀ | Update the dflow binary to the latest release |
| ｀dflow config set-author "Name" --email ...｀ | Store local ｀dflow.author｀ / ｀dflow.email｀ |
| ｀dflow completion [install]｀ | Generate shell completions |
| ｀dflow version｀ | Show the CLI version |

｀dflow finish｀ publishes the work branch to ｀origin｀ before merging, so a
｀manual｀ target's PR can be opened right after with no manual ｀git push｀;
｀--no-push｀ keeps it local.

## Merge mode is the decision authority

｀workflow.branch_rules[].merge_mode｀ in ｀.dflow.yaml｀ decides whether a target
is merged directly by dflow or through a pull request:

| Target merge_mode | Agent action |
|---|---|
| ｀auto｀ | direct merge via ｀dflow finish｀ (clean tree + human confirmation). No PR. |
| ｀manual｀ | open a PR toward that target. Never use ｀dflow finish｀. |

This repository: {{.DefaultMergeMode}} default.{{range .MergeRules}} ｀{{.Branch}} = {{.Mode}}｀.{{end}}

｀auto｀ and ｀manual｀ are the only valid values. An unknown or missing merge mode
makes ｀dflow finish｀ and ｀dflow status｀ fail, naming the branch and the
offending value, instead of silently skipping the target.

Consequences:

- Never suggest a PR toward an ｀auto｀ target; it merges directly.
- Always use a PR toward a ｀manual｀ target (release/hotfix promotions).
- ｀branch_rules｀ entries override ｀default_merge_mode｀ per branch; an unlisted
  branch falls back to the default.

## finish guardrails

｀dflow finish｀ is the exception, not the default. Use it only when:

- the target is ｀auto｀, AND
- the working tree is clean, AND
- a human explicitly confirmed a direct merge (or the repo has no ｀origin｀).

Do not run ｀dflow finish｀ to:

- complete a ｀manual｀ target (open a PR instead), or
- finish a branch that still has stacked children (it would break their merge bases).

Default agent behavior: propose a PR. Direct merge is explicit, never assumed.

## Issue lifecycle

- The repository default branch is ｀{{.DefaultBranch}}｀, so the delivering merge lands
  there. **When** that closes the issue is the trigger in
  "Closing an issue" below; nothing else is a trigger.
- Closing is an explicit step, not a side effect of merging. A ｀Closes #<n>｀
  keyword needs a pull request merged into the default branch to carry it.
- ｀main｀ and release branches do not close issues.
- Issues track any work unit (feature, bug, chore, docs), not only production incidents.

### Closing an issue

Close the issue when the finished branch has no manual targets left, and only then:

- **Trigger** — ｀dflow finish --dry-run｀ prints ｀Manual targets: none｀;
  the machine-readable equivalent is ｀"manual_targets":[]｀ in ｀--json｀.
- **Labels** — remove every ｀status:｀ label naming a state that has ended.
  Keep the issue's ｀type:｀ label.
- **Closing comment** — one comment, naming the merge commit on ｀{{.DefaultBranch}}｀.

## Chained branches

｀dflow start <type> <name> --from <parent>｀ creates a stacked branch.

When a stacked parent merges, rebase each child with
｀git rebase --onto <target> <old-parent> <child>｀ and retarget its PR.

## JSON output is a contract

｀dflow status --json｀, ｀dflow finish --dry-run --json｀ and ｀dflow update --json｀
are contracts: stdout carries exactly one JSON document and nothing else.
A failure exits non-zero with ｀{"error": ...}｀. Parse with a JSON parser.

## Commit conventions

- **Workflow**: Always use ｀dflow｀ to manage branches.
- **Format**: ｀type(scope): description｀ — scope is optional.
- **Types**: ｀feat｀, ｀fix｀, ｀docs｀, ｀style｀, ｀refactor｀, ｀test｀, ｀chore｀.
- **Rules**: English, lowercase, no period, no ticket IDs, no file paths.
`
