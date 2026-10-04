// Package commands defines the end-user subcommands that make up the dflow CLI.
//
// The package contains interactive and non-interactive commands for initializing
// repositories, starting and finishing work branches, deleting branches, and
// managing local dflow metadata stored in Git config.
package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/agent"
	"github.com/yepizrene-devoost/dflow/pkg/flow"
	"github.com/yepizrene-devoost/dflow/pkg/repository"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

// InitCmd initializes the dflow configuration for the current Git project.
//
// This command is intended to be run once per project and will guide the user through
// an interactive setup process to generate a `.dflow.yaml` file with the following:
//
//   - Names for the main, develop, and UAT branches.
//   - Default merge behavior (manual via Pull Requests or automatic).
//   - Optional exceptions for specific branches to use a different merge mode.
//   - Prefixes for feature, release, bugfix and hotfix branches.
//   - Flow rules for each type of branch (feature, release, bugfix, hotfix).
//
// It also ensures the specified base branches exist locally, offering to create them
// if missing. With an `origin` remote configured it asks whether to push those base
// branches there; without one, publication is skipped and the question is not asked.
//
// A repository with no commits cannot have its base branches created, so `init`
// refuses there before asking anything: create the first commit and rerun it.
//
// The `.dflow.yaml` file is stored at the root of the repository and is used by all
// subsequent dflow commands (`start`, `config`, `delete`, etc).
//
// When the project already has a `.dflow.yaml`, `init` refuses to run so a
// hand-edited team contract is never overwritten by accident. Pass `--force` to
// regenerate it deliberately.
//
// Example:
//
//	dflow init
//
// This command is interactive and requires a terminal.
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your dflow branching configuration",
	Long: `Initialize your dflow branching configuration and generate a .dflow.yaml file.

  This interactive setup will guide you through the following steps:

    - Prompt for the names of your main, develop, and UAT branches.
    - Choose your default merge mode and any branch-specific exceptions.
    - Create a .dflow.yaml file with default prefixes:
      - feature/  → for feature branches
      - release/  → for release branches
      - hotfix/   → for hotfix branches
      - bugfix/   → for bugfix branches
    - Set flow rules:
      - Features start from Develop and can be promoted to Develop and UAT
      - Releases start from UAT and can be promoted to Main and Develop
      - Bugfixes start from UAT and sync back to UAT and Develop
      - Hotfixes start from Main and sync back to Main, Develop, and UAT
    - Ensure the specified branches exist locally.
    - Ask whether to push those base branches to origin, when the repository has an
      origin remote that can receive them.

  The resulting .dflow.yaml is stored in the project root and used by all dflow commands.

  A repository with no commits cannot have its base branches created, so init refuses
  there, before asking any question: create the first commit, then rerun init. No
  --force opens that door.

  If .dflow.yaml already exists, init refuses to run instead of overwriting it.
  Pass --force to regenerate the file deliberately. --force only authorizes the
  overwrite: init still runs interactively and requires a terminal.

  Example:
    dflow init
    dflow init --force

  This command is meant to be run once per project when setting up the dflow branching model.`,
	Example: `  dflow init
  dflow init --force`,
	RunE: validators.WithChecks(true, func(cmd *cobra.Command, args []string) error {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		if !force {
			if err := validators.EnsureDflowNotInitialized(); err != nil {
				return err
			}
		}
		ops := defaultInitOperations()
		// The history preflight runs before the terminal check on purpose: a
		// repository with no commits cannot be onboarded at all, so it must hear the
		// real reason rather than be told to open a terminal it is already in, and the
		// wizard must never interrogate the user before failing.
		if err := requireInitRepositoryHistory(ops); err != nil {
			return err
		}
		if !utils.IsInteractive() {
			return fmt.Errorf("dflow init is interactive and requires a terminal")
		}

		draft, err := collectInitDraft()
		if err != nil {
			return err
		}
		draft.force = force
		if err := applyInitDraft(draft, ops); err != nil {
			return err
		}

		utils.Success("Created .dflow.yaml")
		utils.Plain("")
		utils.Success("Merge behavior summary:")
		utils.Plain("   Default mode: %s", draft.config.Workflow.DefaultMergeMode)
		for _, branch := range draft.branches {
			utils.Plain("   - %s: %s", branch, flow.GetMergeModeForBranch(&draft.config, branch))
		}
		utils.Plain("")
		utils.Icon("🎉", "dflow is ready! Use `dflow start` to begin a new branch.")
		return nil
	}),
}

func init() {
	InitCmd.Flags().Bool("force", false, "Regenerate .dflow.yaml even if the project is already initialized")
}

type initDraft struct {
	config        flow.Config
	branches      []string
	push          bool
	generateAgent bool
	force         bool
}

type initRemoteBranch struct {
	exists         bool
	localRevision  string
	remoteRevision string
}

type initOperations struct {
	hasCommits          func() bool
	hasOriginRemote     func() bool
	validateBranch      func(string) error
	ensureBranch        func(string) error
	inspectRemoteBranch func(string) (initRemoteBranch, error)
	pushBranch          func(string) error
	generateAgentFiles  func(*flow.Config) error
	saveConfig          func(*flow.Config) error
}

// The two repository-state guards hold their exact user-facing sentences in one
// place each. The publication line mirrors the wording of the existing
// "Remote 'origin' not found. Skipping pull" line: a step that cannot happen is
// reported as skipped, never as a failure.
const (
	initNoCommitsMessage = "this repository has no commits yet; create the first commit, then rerun `dflow init`"
	initNoOriginMessage  = "Remote 'origin' not found. Skipping base branch publication."
)

// hasInitCommits reports whether HEAD resolves, probed the way the rest of the CLI
// asks the same question: a Git command through the shared capture boundary whose
// success is the whole answer.
//
// An unborn HEAD is not a repository dflow can onboard. `git branch <name>` needs
// an object to point a new branch at, so every base branch would fail with a Git
// diagnostic naming neither the cause nor the fix.
func hasInitCommits() bool {
	session, err := repository.NewSession()
	if err != nil {
		return false
	}
	_, err = gitutils.RunGit(session.Command("rev-parse", "--verify", "--quiet", "HEAD"))
	return err == nil
}

// requireInitRepositoryHistory refuses to onboard a repository with no commits.
//
// It is a preflight and not a hint: `--force` authorizes regenerating .dflow.yaml
// only, so it must not open the door to a run that can only fail later.
func requireInitRepositoryHistory(ops initOperations) error {
	if ops.hasCommits != nil && !ops.hasCommits() {
		return errors.New(initNoCommitsMessage)
	}
	return nil
}

func collectInitDraft() (initDraft, error) {
	var mainBranch, developBranch, uatBranch string
	for _, prompt := range []struct {
		message, defaultValue string
		answer                *string
	}{
		{"Main branch name:", "main", &mainBranch},
		{"Development branch name:", "develop", &developBranch},
		{"UAT branch name:", "uat", &uatBranch},
	} {
		if err := survey.AskOne(&survey.Input{Message: prompt.message, Default: prompt.defaultValue}, prompt.answer, survey.WithValidator(survey.Required)); err != nil {
			return initDraft{}, err
		}
	}

	utils.Plain("")
	utils.Icon("🔧", "Dflow supports two types of merge modes:")
	utils.Plain("   - manual: you open Pull Requests and merge via your platform (e.g. GitHub, GitLab).")
	utils.Plain("   - auto: dflow merges branches directly using Git commands (no PRs needed).")

	var mergeModeOption string
	if err := survey.AskOne(&survey.Select{
		Message: "How do you manage merges by default in this project?",
		Options: []string{"manual (via Pull Requests)", "auto (direct merge from CLI)"},
		Default: "manual (via Pull Requests)",
	}, &mergeModeOption); err != nil {
		return initDraft{}, err
	}
	defaultMode, inverseMode := "manual", "auto"
	if mergeModeOption == "auto (direct merge from CLI)" {
		defaultMode, inverseMode = "auto", "manual"
	}

	cfg := flow.Config{}
	cfg.Branches.Main, cfg.Branches.Develop, cfg.Branches.Uat = mainBranch, developBranch, uatBranch
	cfg.Branches.Features, cfg.Branches.Releases = "feature/", "release/"
	cfg.Branches.Hotfixes, cfg.Branches.Bugfixes = "hotfix/", "bugfix/"
	cfg.Flow.Feature = flow.BranchFlowRule{Base: developBranch, FinishTargets: uniqueBranchNames(developBranch, uatBranch)}
	cfg.Flow.Release = flow.BranchFlowRule{Base: uatBranch, FinishTargets: uniqueBranchNames(mainBranch, developBranch)}
	cfg.Flow.Hotfix = flow.BranchFlowRule{Base: mainBranch, FinishTargets: uniqueBranchNames(mainBranch, developBranch, uatBranch)}
	cfg.Flow.Bugfix = flow.BranchFlowRule{Base: uatBranch, FinishTargets: uniqueBranchNames(uatBranch, developBranch)}
	cfg.Workflow.DefaultMergeMode = defaultMode
	cfg.Workflow.BranchRules = make(map[string]flow.WorkflowBranchRule)

	branches := uniqueBranchNames(mainBranch, developBranch, uatBranch)
	var exceptionBranches []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: fmt.Sprintf("Which branches should behave differently from the default '%s' mode?", defaultMode),
		Options: branches,
		Help:    fmt.Sprintf("Select the branches that require '%s' instead of the default '%s'", inverseMode, defaultMode),
	}, &exceptionBranches); err != nil {
		return initDraft{}, err
	}
	for _, branch := range branches {
		cfg.Workflow.BranchRules[branch] = flow.WorkflowBranchRule{MergeMode: defaultMode}
	}
	for _, branch := range exceptionBranches {
		cfg.Workflow.BranchRules[branch] = flow.WorkflowBranchRule{MergeMode: inverseMode}
	}

	draft := initDraft{config: cfg, branches: branches}
	// The publication question is only offered where its answer can be acted on:
	// its default is "yes", so asking it in a repository without an 'origin' remote
	// invited the user to abort their own onboarding.
	if gitutils.HasOriginRemote() {
		if err := survey.AskOne(&survey.Confirm{Message: "Do you want to push the base branches to 'origin'?", Default: true}, &draft.push); err != nil {
			return initDraft{}, fmt.Errorf("push prompt failed: %w", err)
		}
	}
	if err := survey.AskOne(&survey.Confirm{Message: "Generate an agent workflow file for AI coding assistants?", Default: true}, &draft.generateAgent); err != nil {
		return initDraft{}, fmt.Errorf("agent workflow prompt failed: %w", err)
	}
	return draft, nil
}

func defaultInitOperations() initOperations {
	return initOperations{
		hasCommits:          hasInitCommits,
		hasOriginRemote:     gitutils.HasOriginRemote,
		validateBranch:      validateInitBranch,
		ensureBranch:        gitutils.CheckOrCreateBranch,
		inspectRemoteBranch: inspectInitRemoteBranch,
		pushBranch:          gitutils.PushBranch,
		generateAgentFiles:  generateInitAgentFiles,
		saveConfig:          utils.SaveConfig,
	}
}

func applyInitDraft(draft initDraft, ops initOperations) error {
	if err := requireInitRepositoryHistory(ops); err != nil {
		return err
	}
	if err := draft.config.Validate(); err != nil {
		return fmt.Errorf("validate onboarding configuration: %w", err)
	}
	primary := []struct{ role, name string }{
		{"main", draft.config.Branches.Main},
		{"development", draft.config.Branches.Develop},
		{"UAT", draft.config.Branches.Uat},
	}
	for _, branch := range primary {
		if err := ops.validateBranch(branch.name); err != nil {
			return fmt.Errorf("invalid %s branch %q: %w", branch.role, branch.name, err)
		}
	}

	var local, remote []string
	for _, branch := range draft.branches {
		if err := ops.ensureBranch(branch); err != nil {
			return initProgressError(draft.force, fmt.Sprintf("create or reuse local branch %q", branch), err, local, remote)
		}
		local = append(local, branch)
	}
	if !initHasOriginRemote(ops) {
		// One report per run, emitted from this single place: the wizard never asks
		// about publication without a remote, so no other path can print it. The
		// guard still covers a requested publication, because an unguarded
		// `git ls-remote --heads origin` in a repository without that remote is what
		// aborted onboarding.
		utils.Icon("📁", "%s", initNoOriginMessage)
	} else if draft.push {
		for _, branch := range draft.branches {
			state, err := ops.inspectRemoteBranch(branch)
			if err != nil {
				return initProgressError(draft.force, fmt.Sprintf("check remote branch %q", branch), err, local, remote)
			}
			if state.exists {
				remote = append(remote, branch)
				if state.localRevision != state.remoteRevision {
					cause := fmt.Errorf("local revision %s differs from remote revision %s; refusing to overwrite the existing remote branch", state.localRevision, state.remoteRevision)
					return initProgressError(draft.force, fmt.Sprintf("verify existing remote branch %q", branch), cause, local, remote)
				}
				continue
			}
			if err := ops.pushBranch(branch); err != nil {
				return initProgressError(draft.force, fmt.Sprintf("publish branch %q", branch), err, local, remote)
			}
			remote = append(remote, branch)
		}
	}
	if draft.generateAgent && ops.generateAgentFiles != nil {
		if err := ops.generateAgentFiles(&draft.config); err != nil {
			return initProgressError(draft.force, "generate agent workflow", err, local, remote)
		}
	}
	if err := ops.saveConfig(&draft.config); err != nil {
		return initProgressError(draft.force, "write final .dflow.yaml", err, local, remote)
	}
	return nil
}

// initHasOriginRemote answers the publication guard through the injectable
// operation. An unwired probe means "the remote exists": like generateAgentFiles,
// a missing operation must not fabricate a failure.
func initHasOriginRemote(ops initOperations) bool {
	return ops.hasOriginRemote == nil || ops.hasOriginRemote()
}

func validateInitBranch(branch string) error {
	output, err := exec.CommandContext(context.Background(), "git", "check-ref-format", "--branch", branch).CombinedOutput()
	if err == nil {
		return nil
	}
	diagnostic := strings.TrimSpace(string(output))
	if diagnostic == "" {
		return err
	}
	return fmt.Errorf("%s: %w", diagnostic, err)
}

func inspectInitRemoteBranch(branch string) (initRemoteBranch, error) {
	session, err := repository.NewSession()
	if err != nil {
		return initRemoteBranch{}, fmt.Errorf("discover repository for remote inspection: %w", err)
	}
	localRevision, err := gitutils.RunGit(session.Command("rev-parse", "refs/heads/"+branch))
	if err != nil {
		return initRemoteBranch{}, fmt.Errorf("read local revision for %q: %w", branch, err)
	}
	remoteOutput, err := gitutils.RunGit(session.Command("ls-remote", "--heads", "origin", "refs/heads/"+branch))
	if err != nil {
		return initRemoteBranch{}, fmt.Errorf("read remote revision for %q: %w", branch, err)
	}
	state := initRemoteBranch{localRevision: strings.TrimSpace(localRevision)}
	fields := strings.Fields(remoteOutput)
	if len(fields) == 0 {
		return state, nil
	}
	state.exists = true
	state.remoteRevision = fields[0]
	return state, nil
}

func initProgressError(force bool, stage string, cause error, local, remote []string) error {
	localState := "No local branches were confirmed"
	if len(local) > 0 {
		localState = "Local branches retained: " + strings.Join(local, ", ")
	}
	remoteState := "No remote branches were published"
	if len(remote) > 0 {
		remoteState = "Remote branches retained: " + strings.Join(remote, ", ")
	}
	retryCommand := "dflow init"
	if force {
		retryCommand += " --force"
	}
	return fmt.Errorf("onboarding stopped while attempting to %s: %w. %s. %s. .dflow.yaml was not changed; resolve the reported error and rerun `%s` to reuse existing refs", stage, cause, localState, remoteState, retryCommand)
}

func generateInitAgentFiles(cfg *flow.Config) error {
	doc := agent.GenerateAgentDoc(cfg)
	agentPath := ".agents/workflows/dflow.md"
	if err := os.MkdirAll(filepath.Dir(agentPath), 0755); err != nil {
		return fmt.Errorf("failed to create agent workflow directory: %w", err)
	}
	if err := os.WriteFile(agentPath, doc, 0644); err != nil {
		return fmt.Errorf("failed to write agent workflow: %w", err)
	}
	utils.Success("Generated agent workflow: %s", agentPath)
	plan := agent.PlanInstructionTargets(agent.Agents(), instructionFileExists, false)
	if err := writeInstructionReferences(plan, agentPath); err != nil {
		utils.Warn("Could not update instruction references: %v", err)
	}
	return nil
}

func uniqueBranchNames(branches ...string) []string {
	var unique []string
	for _, branch := range branches {
		if branch == "" || slices.Contains(unique, branch) {
			continue
		}
		unique = append(unique, branch)
	}
	return unique
}
