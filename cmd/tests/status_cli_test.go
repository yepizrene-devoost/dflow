package tests

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/flow"
)

// TestStatusAndFinishJSONCLI covers the machine-readable contract a
// non-interactive caller depends on, against the real binary: exactly one JSON
// document and nothing else on stdout, a non-zero exit on failure, and a
// merge_mode that fails loudly instead of silently skipping a target.
func TestStatusAndFinishJSONCLI(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	// Subprocesses inherit this process environment. Clearing DFLOW_CWD makes
	// the binary load the .dflow.yaml of the repository it is run in, even if an
	// in-process test set the variable earlier.
	t.Setenv("DFLOW_CWD", "")

	binary := buildDflowCLI(t)

	t.Run("status --json", func(t *testing.T) {
		repo := setupStatusRepo(t, statusTestConfig(), "feature/json-status")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "status", "--json")
		if exitCode != 0 {
			t.Fatalf("status --json exited %d, want 0\n%s", exitCode, output)
		}
		assertNoHumanChrome(t, output)

		doc := decodeSingleJSONDocument(t, output)
		requireJSONString(t, doc, "branch", "feature/json-status")
		requireJSONString(t, doc, "branch_type", "feature")
		requireJSONString(t, doc, "base", "develop")
		requireJSONBool(t, doc, "working_tree_clean", true)
		requireJSONBool(t, doc, "merge_in_progress", false)
		requireJSONBool(t, doc, "has_origin", true)

		targets := requireJSONArray(t, doc, "targets")
		if len(targets) != 2 {
			t.Fatalf("targets has %d entries, want 2:\n%v", len(targets), targets)
		}
		first, ok := targets[0].(map[string]any)
		if !ok {
			t.Fatalf("targets[0] = %#v, want an object", targets[0])
		}
		if first["branch"] != "develop" {
			t.Fatalf("targets[0].branch = %v, want develop", first["branch"])
		}
		if first["merge_mode"] != "auto" {
			t.Fatalf("targets[0].merge_mode = %v, want auto", first["merge_mode"])
		}
	})

	t.Run("status --json on a non-work branch", func(t *testing.T) {
		repo := setupStatusRepo(t, statusTestConfig(), "")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "status", "--json")
		if exitCode != 0 {
			t.Fatalf("status --json on a non-work branch exited %d, want 0\n%s", exitCode, output)
		}
		assertNoHumanChrome(t, output)

		doc := decodeSingleJSONDocument(t, output)
		requireJSONString(t, doc, "branch", "main")
		requireJSONString(t, doc, "branch_type", "")
		requireJSONString(t, doc, "base", "")

		targets := requireJSONArray(t, doc, "targets")
		if len(targets) != 0 {
			t.Fatalf("targets = %v, want an empty array", targets)
		}
	})

	t.Run("finish --dry-run --json", func(t *testing.T) {
		repo := setupStatusRepo(t, statusTestConfig(), "feature/json-status")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "finish", "--dry-run", "--json")
		if exitCode != 0 {
			t.Fatalf("finish --dry-run --json exited %d, want 0\n%s", exitCode, output)
		}
		assertNoHumanChrome(t, output)

		doc := decodeSingleJSONDocument(t, output)
		requireJSONString(t, doc, "current_branch", "feature/json-status")
		requireJSONString(t, doc, "branch_type", "feature")
		requireJSONString(t, doc, "base", "develop")
		requireJSONString(t, doc, "return_branch", "develop")
		requireJSONBool(t, doc, "delete_requested", false)
		requireJSONBool(t, doc, "publish_work_branch", true)
		requireJSONBool(t, doc, "dry_run", true)

		targets := requireJSONArray(t, doc, "targets")
		if len(targets) != 2 {
			t.Fatalf("targets has %d entries, want 2:\n%v", len(targets), targets)
		}
		requireJSONStringSlice(t, doc, "auto_targets", []string{"develop"})
		requireJSONStringSlice(t, doc, "manual_targets", []string{"main"})
	})

	t.Run("finish --dry-run --json --no-push", func(t *testing.T) {
		repo := setupStatusRepo(t, statusTestConfig(), "feature/json-status")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "finish", "--dry-run", "--json", "--no-push")
		if exitCode != 0 {
			t.Fatalf("finish --dry-run --json --no-push exited %d, want 0\n%s", exitCode, output)
		}
		assertNoHumanChrome(t, output)

		doc := decodeSingleJSONDocument(t, output)
		requireJSONBool(t, doc, "publish_work_branch", false)
		requireJSONBool(t, doc, "dry_run", true)
		requireJSONStringSlice(t, doc, "auto_targets", []string{"develop"})
		requireJSONStringSlice(t, doc, "manual_targets", []string{"main"})
	})

	t.Run("finish --json requires --dry-run", func(t *testing.T) {
		repo := setupStatusRepo(t, statusTestConfig(), "feature/json-status")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "finish", "--json")
		if exitCode == 0 {
			t.Fatalf("finish --json without --dry-run exited 0, want non-zero\n%s", output)
		}
		assertNoHumanChrome(t, output)

		// The rejection is itself a JSON document, because --json was requested.
		doc := decodeSingleJSONDocument(t, output)
		message, ok := doc["error"].(string)
		if !ok {
			t.Fatalf("failure document has no string \"error\" key:\n%v", doc)
		}
		if !strings.Contains(message, "--dry-run") {
			t.Fatalf("failure message must say --dry-run is required, got:\n%s", message)
		}
	})

	t.Run("invalid merge mode", func(t *testing.T) {
		cfg := statusTestConfig()
		cfg.Workflow.BranchRules["develop"] = flow.WorkflowBranchRule{MergeMode: "pr"}
		repo := setupStatusRepo(t, cfg, "feature/json-status")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "finish", "--dry-run")
		if exitCode == 0 {
			t.Fatalf("finish --dry-run with an invalid merge mode exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "pr") {
			t.Fatalf("failure message must name the offending value \"pr\":\n%s", output)
		}
		if !strings.Contains(output, "develop") {
			t.Fatalf("failure message must name the branch \"develop\":\n%s", output)
		}
	})

	t.Run("unset merge mode", func(t *testing.T) {
		cfg := statusTestConfig()
		cfg.Workflow.DefaultMergeMode = ""
		delete(cfg.Workflow.BranchRules, "develop")
		repo := setupStatusRepo(t, cfg, "feature/json-status")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "finish", "--dry-run")
		if exitCode == 0 {
			t.Fatalf("finish --dry-run with an unset merge mode exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "workflow.default_merge_mode") {
			t.Fatalf("failure message must point at the unset configuration:\n%s", output)
		}
	})

	t.Run("human output", func(t *testing.T) {
		repo := setupStatusRepo(t, statusTestConfig(), "feature/json-status")

		statusOutput, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "status")
		if exitCode != 0 {
			t.Fatalf("status exited %d, want 0\n%s", exitCode, statusOutput)
		}
		for _, line := range []string{
			"branch: feature/json-status\n",
			"branch_type: feature\n",
			"base: develop\n",
			"targets: develop (auto), main (manual)\n",
			"working_tree_clean: true\n",
			"merge_in_progress: false\n",
			"has_origin: true\n",
		} {
			if !strings.Contains(statusOutput, line) {
				t.Fatalf("human status output is missing %q:\n%s", line, statusOutput)
			}
		}

		finishOutput, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "finish", "--dry-run")
		if exitCode != 0 {
			t.Fatalf("finish --dry-run exited %d, want 0\n%s", exitCode, finishOutput)
		}
		for _, line := range []string{
			"Dry run enabled. No branches will be checked out, merged, or pushed.\n",
			"Would return to branch: develop\n",
			"Would merge 'feature/json-status' into 'develop' and push the target branch.\n",
		} {
			if !strings.Contains(finishOutput, line) {
				t.Fatalf("human dry-run output is missing %q:\n%s", line, finishOutput)
			}
		}
	})
}

// TestJSONFlagParseFailureStaysJSON guards the "JSON in, JSON out" contract at
// the earliest failure point. Cobra parses flags and validates args before any
// command's PreRunE runs, so a flag-parse or arity error on a --json invocation
// used to fall through to the human renderer. The format is now selected before
// RootCmd.Execute, so the failure is still exactly one parseable JSON document.
func TestJSONFlagParseFailureStaysJSON(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	t.Setenv("DFLOW_CWD", "")

	binary := buildDflowCLI(t)

	for _, args := range [][]string{
		{"status", "--json", "--bogus-flag"},
		{"status", "--json=true", "--bogus-flag"},
		{"status", "--json", "unexpected-argument"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			// A flag-parse failure must not depend on a repository or config: the
			// format decision comes from the raw arguments alone.
			output, exitCode := startCLIRawOutput(t, 15*time.Second, t.TempDir(), binary, args...)
			if exitCode == 0 {
				t.Fatalf("%v exited 0, want non-zero\n%s", args, output)
			}
			assertNoHumanChrome(t, output)

			doc := decodeSingleJSONDocument(t, output)
			message, ok := doc["error"].(string)
			if !ok || message == "" {
				t.Fatalf("failure document has no non-empty string \"error\" key:\n%v", doc)
			}
		})
	}
}

// buildDflowCLI returns the real entry point as a fresh copy of the package's
// shared, memoized plain build (no linker flag, so the binary reports the
// literal "dev" marker), so Cobra parsing, PreRunE ordering and the exit code
// are all covered by the assertions without paying a compile per test.
func buildDflowCLI(t *testing.T) string {
	t.Helper()

	return sharedDflowCLI(t, "")
}

// statusTestConfig is a realistic two-target config: develop merges directly and
// main is left for PR flow.
func statusTestConfig() *flow.Config {
	cfg := &flow.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = []string{"develop", "main"}
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
		"main":    {MergeMode: "manual"},
	}
	return cfg
}

// setupStatusRepo creates a repository with an origin remote, the given config
// committed (so the working tree is clean) and, when branch is non-empty, that
// branch checked out.
func setupStatusRepo(t *testing.T, cfg *flow.Config, branch string) string {
	t.Helper()

	repo := initTempGitRepo(t)
	if branch != "" {
		runGit(t, repo, "checkout", "-b", branch)
	}

	remote := initBareGitRepo(t)
	runGit(t, repo, "remote", "add", "origin", remote)

	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(cfg); err != nil {
			t.Fatalf("save status fixture config: %v", err)
		}
	})
	runGit(t, repo, "add", ".dflow.yaml")
	runGit(t, repo, "commit", "-m", "add dflow config")

	return repo
}

// decodeSingleJSONDocument parses output as exactly one JSON object and rejects
// trailing content, so a banner or a second document cannot pass unnoticed.
func decodeSingleJSONDocument(t *testing.T, output string) map[string]any {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(output))
	var doc map[string]any
	if err := decoder.Decode(&doc); err != nil {
		t.Fatalf("stdout is not a JSON document: %v\noutput:\n%s", err, output)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("stdout must contain exactly one JSON document, found trailing content (%v):\n%s", err, output)
	}

	return doc
}

// assertNoHumanChrome asserts that a machine-readable stream carries no banner,
// no carriage return, no ANSI escape and no status icon.
func assertNoHumanChrome(t *testing.T, output string) {
	t.Helper()

	if strings.ContainsRune(output, '\r') {
		t.Fatalf("machine-readable output contains a carriage return:\n%q", output)
	}
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("machine-readable output contains an ANSI escape:\n%q", output)
	}
	if strings.Contains(output, "Git branching made simple") {
		t.Fatalf("machine-readable output contains the banner:\n%s", output)
	}
	if strings.Contains(output, "\u274c") || strings.Contains(output, "\u2705") {
		t.Fatalf("machine-readable output contains a status icon:\n%s", output)
	}
}

func requireJSONString(t *testing.T, doc map[string]any, key, want string) {
	t.Helper()

	value, ok := doc[key]
	if !ok {
		t.Fatalf("JSON document is missing key %q:\n%v", key, doc)
	}
	got, ok := value.(string)
	if !ok {
		t.Fatalf("key %q = %#v, want a string", key, value)
	}
	if got != want {
		t.Fatalf("key %q = %q, want %q", key, got, want)
	}
}

func requireJSONBool(t *testing.T, doc map[string]any, key string, want bool) {
	t.Helper()

	value, ok := doc[key]
	if !ok {
		t.Fatalf("JSON document is missing key %q:\n%v", key, doc)
	}
	got, ok := value.(bool)
	if !ok {
		t.Fatalf("key %q = %#v, want a boolean", key, value)
	}
	if got != want {
		t.Fatalf("key %q = %t, want %t", key, got, want)
	}
}

// requireJSONArray asserts the key is present and, deliberately, is not null.
func requireJSONArray(t *testing.T, doc map[string]any, key string) []any {
	t.Helper()

	value, ok := doc[key]
	if !ok {
		t.Fatalf("JSON document is missing key %q:\n%v", key, doc)
	}
	if value == nil {
		t.Fatalf("key %q is null, want an array", key)
	}
	array, ok := value.([]any)
	if !ok {
		t.Fatalf("key %q = %#v, want an array", key, value)
	}
	return array
}

func requireJSONStringSlice(t *testing.T, doc map[string]any, key string, want []string) {
	t.Helper()

	array := requireJSONArray(t, doc, key)
	if len(array) != len(want) {
		t.Fatalf("key %q = %v, want %v", key, array, want)
	}
	for i, expected := range want {
		got, ok := array[i].(string)
		if !ok {
			t.Fatalf("key %q[%d] = %#v, want a string", key, i, array[i])
		}
		if got != expected {
			t.Fatalf("key %q[%d] = %q, want %q", key, i, got, expected)
		}
	}
}
