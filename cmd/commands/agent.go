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

Pass --install to install the document as a discoverable agent skill as well: the
same bytes, written as SKILL.md in the skills directory each selected agent
reads. Auto targets the portable .agents/skills root — one file that pi, codex and
opencode all discover — because the private per-agent roots are opt-in: name
agents, or "all", to widen the install to their own directories. --install
--local keeps the skill inside the project instead of the user's home, and --local
on its own is an error, because it only selects the scope of an installation.
An install never needs --force: an existing document is left untouched and only
the skill is generated, so a document a team has hand-edited can differ from the
installed skill.

Pass --json to receive the target path, the rendered document, the reference plan
and the skill plan as a single machine-readable document instead of writing
anything. --json is read-only: it writes neither the document nor any reference
nor any skill, and an existing document is not an error for it.`,
	Example: `  dflow agent
  dflow agent --path docs/AGENT.md
  dflow agent --force
  dflow agent --agents claude
  dflow agent --install
  dflow agent --install --local
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
		install, err := cmd.Flags().GetBool("install")
		if err != nil {
			return err
		}
		local, err := cmd.Flags().GetBool("local")
		if err != nil {
			return err
		}

		// --local only chooses the scope an install writes to, so on its own it is a
		// request with no effect. Refusing it here — before anything is written —
		// keeps `dflow agent --local` from looking like a run that installed
		// something where the user asked.
		if local && !install {
			return fmt.Errorf("--local requires --install: it selects the scope of a skill installation")
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
		skills := agent.PlanSkillPlacements(skillPlacementAgents(agents, explicit), local)

		if utils.CurrentFormat() == utils.FormatJSON {
			// --json is strictly read-only: it renders the document and reports
			// the plans it would apply, and writes nothing at all. The existence
			// check below belongs to the write path, so an existing document must
			// not make a read-only render refuse to render.
			return utils.EmitJSON(agentJSONReport(absPath, doc, plan, skills))
		}

		// The document is derived and a hand-edited copy is protected: it is only
		// rewritten under --force. An existing document therefore refuses the run,
		// UNLESS --install asked for the skill as well — the skill is generated from
		// the bytes this run rendered, so installing it needs no overwrite of the
		// document, and pushing the user onto the destructive --force path to install
		// a skill would be exactly backwards. The document on disk and the installed
		// skill can then differ, which is intended: the document is protected, the
		// skill is generated.
		documentExists := false
		if _, statErr := os.Stat(path); statErr == nil {
			documentExists = true
		} else if !os.IsNotExist(statErr) {
			return statErr
		}

		if documentExists && !force && !install {
			return fmt.Errorf("%s already exists; use --force to overwrite it", path)
		}

		if documentExists && !force {
			// An install over a protected document: report the document truthfully,
			// because this run does not write it. The skill below still receives the
			// freshly rendered bytes.
			utils.Info("Left the existing agent workflow document untouched: %s", path)
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", path, err)
			}

			if err := os.WriteFile(path, doc, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", path, err)
			}

			utils.Success("Generated agent workflow: %s", path)
		}

		if err := writeInstructionReferences(plan, path); err != nil {
			return err
		}
		if !install {
			return nil
		}
		return installSkills(skills, doc)
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

// skillPlacementAgents resolves the agents whose skill directories an install
// targets, from the same --agents selection the instruction references use.
//
// Auto (no --agents) targets the PORTABLE artifact, which is this command's
// default on both discovery surfaces: the instruction side already targets
// AGENTS.md, the one file pi, codex and opencode all read, and the skill side
// targets the shared .agents/skills root those same three agents discover. Issue
// #34 names exactly that path for a plain `dflow agent --install`, one file
// serving several agents is the point of the work unit, and the private per-agent
// roots (~/.codex/skills, ~/.config/opencode/skills, ~/.claude/skills) stay opt-in
// so an unattended run never scatters copies through $HOME. Naming agents —
// `--agents claude`, or "all" — widens the install to their own roots.
//
// The portable agent is looked up in the registry rather than spelled out here,
// so correcting the table corrects this too.
func skillPlacementAgents(agents []agent.Agent, explicit bool) []agent.Agent {
	if explicit {
		return agents
	}
	portable, ok := agent.LookupAgent(agent.AgentPi)
	if !ok {
		// A registry without the portable agent is a table defect. Falling back to
		// the full selection keeps the command working, and the resulting plan is
		// visible in --json rather than silent.
		return agents
	}
	return []agent.Agent{portable}
}

// installSkills writes the rendered document as a discoverable skill into every
// planned directory, reporting one line per placement.
//
// The bytes are the document's own, never a second rendering: the skill is the
// same artifact an agent reaches through the instruction reference, so the two
// cannot disagree. A file that is already current is reported as such instead of
// claiming a write dflow did not make.
func installSkills(placements []agent.SkillPlacement, doc []byte) error {
	for _, placement := range placements {
		changed, path, err := agent.InstallSkill(placement.Dir, doc)
		if err != nil {
			return err
		}
		if changed {
			utils.Success("Installed the %s skill: %s", agent.SkillName, path)
			continue
		}
		utils.Success("The %s skill is already current: %s", agent.SkillName, path)
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

// agentSkillReport is one planned skill placement in the --json report: the
// skills root as the registry records it, the directory the file goes into, the
// agents that discover it, and whether the file is already there.
//
// Exists is a fact about the filesystem, which a read-only render can check
// without writing. There is deliberately no "changed" field, for the same reason
// the reference plan carries none: whether a file would change cannot be known
// without writing it.
type agentSkillReport struct {
	Root   string   `json:"root"`
	Dir    string   `json:"dir"`
	Agents []string `json:"agents"`
	Exists bool     `json:"exists"`
}

// agentJSONReport builds the machine-readable result of `dflow agent --json`: the
// target path and the rendered document, unchanged from before these plans
// existed, plus the reference plan and the skill plan the run would apply.
func agentJSONReport(absPath string, doc []byte, plan []agent.InstructionTarget, placements []agent.SkillPlacement) map[string]any {
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

	skills := make([]agentSkillReport, 0, len(placements))
	for _, placement := range placements {
		agents := make([]string, 0, len(placement.Agents))
		for _, id := range placement.Agents {
			agents = append(agents, string(id))
		}
		skills = append(skills, agentSkillReport{
			Root:   placement.Root,
			Dir:    placement.Dir,
			Agents: agents,
			Exists: skillFileExists(placement.Dir),
		})
	}

	return map[string]any{
		"path":       absPath,
		"content":    string(doc),
		"references": references,
		"skills":     skills,
	}
}

// skillFileExists reports whether a planned skill file is already installed.
//
// A directory whose path cannot be resolved is reported as not having one: this
// is a read-only report, so an unreadable location is a location with no file it
// can confirm, never a write and never a guess.
func skillFileExists(dir string) bool {
	path, err := agent.SkillFilePath(dir)
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func init() {
	AgentCmd.Flags().String("path", ".agents/workflows/dflow.md", "Path where the agent workflow document is written")
	AgentCmd.Flags().Bool("force", false, "Overwrite the target file when it already exists")
	AgentCmd.Flags().String("agents", "", `Agents to wire the workflow reference for: a comma-separated id list, or "all" (default: the portable default — AGENTS.md and the shared .agents/skills root; name agents, or "all", to widen it to their own roots)`)
	AgentCmd.Flags().Bool("install", false, "Also install the workflow document as a discoverable agent skill, with the same bytes")
	AgentCmd.Flags().Bool("local", false, "With --install, install into the project's skills directory instead of the user's")
	AgentCmd.Flags().Bool("json", false, "Print the target path, the rendered document, the reference plan and the skill plan as a single JSON document on stdout, writing nothing")
}
