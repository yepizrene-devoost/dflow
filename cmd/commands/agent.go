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
team has hand-edited is not silently replaced. Pass --json to receive the target
path and the rendered document as a single machine-readable document instead of
writing it, which is useful for callers that manage the file themselves.`,
	Example: `  dflow agent
  dflow agent --path docs/AGENT.md
  dflow agent --force`,
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

		cfg, err := utils.LoadConfig()
		if err != nil {
			return err
		}

		doc := agent.GenerateAgentDoc(cfg)

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
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

		if utils.CurrentFormat() == utils.FormatJSON {
			return utils.EmitJSON(map[string]string{
				"path":    absPath,
				"content": string(doc),
			})
		}

		utils.Success("Generated agent workflow: %s", path)
		return nil
	}),
}

func init() {
	AgentCmd.Flags().String("path", ".agents/workflows/dflow.md", "Path where the agent workflow document is written")
	AgentCmd.Flags().Bool("force", false, "Overwrite the target file when it already exists")
	AgentCmd.Flags().Bool("json", false, "Print the target path and rendered document as a single JSON document on stdout")
}
