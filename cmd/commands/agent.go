package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/agent"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

// AgentCmd renders the project's agent workflow document from `.dflow.yaml`.
//
// The document is derived, never hand-maintained: it restates the configured
// branch types, finish targets and merge modes so an agent working in the
// repository reads the same rules dflow enforces. Regenerating it is idempotent
// as long as the configuration has not changed.
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Generate an agent workflow file from .dflow.yaml",
	Long: `Generate an agent workflow document from the project's .dflow.yaml.

The command renders the configured branch types, their prefixes, bases and
finish targets, the per-branch merge modes, and the workflow rules an agent must
follow, then writes the result as markdown. The output is written to
.agents/workflows/dflow.md by default; pass --path to choose another location
(for example docs/AGENT.md).

An existing file is never overwritten unless --force is passed, so a document a
team has hand-edited is not silently replaced.

After writing the document the command points the instruction files the
supported agents read at it: AGENTS.md — read by pi, codex and opencode — is
always ensured and created when missing, while CLAUDE.md is kept current only
when the project already has one, so no run drops a surprise CLAUDE.md into a
repository that does not use Claude Code. Pass --agents to name the agents
yourself, either as a comma-separated id list or as "all": naming claude creates
CLAUDE.md, and naming claude alone leaves AGENTS.md alone.

Pass --json to receive the target path, the rendered document and the reference
plan as a single machine-readable document instead of writing anything. --json
is read-only: it writes neither the document nor any reference, and an existing
document is not an error for it.`,
	Example: `  dflow agent
  dflow agent --path docs/AGENT.md
  dflow agent --force
  dflow agent --agents claude
  dflow agent --json`,
	Args: cobra.NoArgs,
	// The output format has to be decided before the WithChecks wrapper inside
	// RunE runs, so a failure such as a missing .dflow.yaml is reported in the
	// requested format instead of as styled text. Cobra runs PersistentPreRun,
	// then PreRunE, then RunE.
	PreRunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}
		if jsonOutput {
			utils.SetFormat(utils.FormatJSON)
		}
		return nil
	},
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			return err
		}
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		spec, err := cmd.Flags().GetString("agents")
		if err != nil {
			return err
		}

		// The selection is resolved before anything is written: an unknown agent
		// id must fail without leaving a document, a directory or a reference
		// behind, so `--agents bogus` cannot half-apply.
		agents, err := agent.ParseAgentSpec(spec)
		if err != nil {
			return err
		}
		// Auto versus named is read from the spec itself, not from the flag's
		// Changed bit: "" is documented as auto, so an explicit `--agents ""`
		// must mean exactly what an absent --agents means instead of silently
		// turning into a named "every agent" selection (which would create a
		// CLAUDE.md nobody asked for).
		explicit := spec != ""

		cfg, err := utils.LoadConfig()
		if err != nil {
			return err
		}

		doc := agent.GenerateAgentDoc(cfg)

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		plan := agent.PlanInstructionTargets(agents, instructionFileExists, explicit)

		if utils.CurrentFormat() == utils.FormatJSON {
			// --json is strictly read-only: it renders the document and reports
			// the plan it would apply, and writes nothing at all. The existence
			// check below belongs to the write path, so an existing document must
			// not make a read-only render refuse to render.
			return utils.EmitJSON(agentJSONReport(absPath, doc, plan))
		}

		if !force {
			if _, statErr := os.Stat(path); statErr == nil {
				return fmt.Errorf("%s already exists; use --force to overwrite it", path)
			} else if !os.IsNotExist(statErr) {
				return statErr
			}
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}

		if err := os.WriteFile(path, doc, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}

		utils.Success("Generated agent workflow: %s", path)

		return writeInstructionReferences(plan, path)
	}),
}

// instructionFileExists reports whether a project-relative instruction file
// exists, which is the whole fact the planner needs to choose between updating
// a file and creating it.
func instructionFileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

// writeInstructionReferences makes every planned instruction file carry the
// reference to workflowPath, reporting one line per file through the shared
// output helper.
//
// workflowPath is the project-relative path the caller actually wrote, never the
// absolute form: the reference is committed inside an instruction file, where an
// absolute machine path would be meaningless.
func writeInstructionReferences(plan []agent.InstructionTarget, workflowPath string) error {
	for _, target := range plan {
		changed, err := agent.EnsureInstructionReference(target.File, workflowPath)
		if err != nil {
			return err
		}
		if changed {
			utils.Success("Updated %s with dflow workflow reference", target.File)
			continue
		}
		utils.Success("%s already references the dflow workflow", target.File)
	}
	return nil
}

// agentReferenceReport is one planned instruction file in the --json report:
// the file, the registry agents that read it, and whether this run would create
// it.
//
// There is deliberately no "changed" field. Whether a reference file would change
// cannot be known without writing it, and --json writes nothing, so reporting one
// would be a guess dressed as a fact.
type agentReferenceReport struct {
	File   string   `json:"file"`
	Agents []string `json:"agents"`
	Create bool     `json:"create"`
}

// agentJSONReport builds the machine-readable result of `dflow agent --json`: the
// target path and the rendered document, unchanged from before this mode grew a
// plan, plus the reference plan the run would apply.
func agentJSONReport(absPath string, doc []byte, plan []agent.InstructionTarget) map[string]any {
	references := make([]agentReferenceReport, 0, len(plan))
	for _, target := range plan {
		agents := make([]string, 0, len(target.Agents))
		for _, id := range target.Agents {
			agents = append(agents, string(id))
		}
		references = append(references, agentReferenceReport{
			File:   target.File,
			Agents: agents,
			Create: target.Create,
		})
	}
	return map[string]any{
		"path":       absPath,
		"content":    string(doc),
		"references": references,
	}
}

func init() {
	AgentCmd.Flags().String("path", ".agents/workflows/dflow.md", "Path where the agent workflow document is written")
	AgentCmd.Flags().Bool("force", false, "Overwrite the target file when it already exists")
	AgentCmd.Flags().String("agents", "", `Agents to wire the workflow reference for: a comma-separated id list, or "all" (default: every supported agent, with CLAUDE.md only when it already exists)`)
	AgentCmd.Flags().Bool("json", false, "Print the target path, the rendered document and the reference plan as a single JSON document on stdout, writing nothing")
}
