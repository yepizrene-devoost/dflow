package commands

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/pkg/flow"
)

// initNoCommitsWording and initPublicationSkipWording are the exact user-facing
// sentences the two new guards must emit. They are spelled out here, rather than
// referenced from the implementation, so the tests pin the wording itself: a
// reworded message has to be a deliberate change to these literals.
const (
	initNoCommitsWording       = "this repository has no commits yet; create the first commit, then rerun `dflow init`"
	initPublicationSkipWording = "Remote 'origin' not found. Skipping base branch publication."
)

// TestInitApplyAbortsCommitlessRepositoryBeforeMutation pins the WU1 guard at the
// apply path: a repository with no commits cannot have base branches created, so
// the run must refuse before any step that could mutate anything.
func TestInitApplyAbortsCommitlessRepositoryBeforeMutation(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	draft.generateAgent = true
	ops, calls := recordingInitOperations()
	ops.hasCommits = func() bool { return false }

	err := applyInitDraft(draft, ops)
	if err == nil || err.Error() != initNoCommitsWording {
		t.Fatalf("applyInitDraft() error = %v, want the exact refusal %q", err, initNoCommitsWording)
	}
	assertNoInitMutations(t, *calls)
	if len(*calls) != 0 {
		t.Fatalf("commitless repository reached onboarding steps: %v", *calls)
	}
}

// TestInitApplySkipsPublicationWithoutOrigin pins WU2: publication is dropped
// with the informational line, no remote lookup and no push are attempted (the
// old unguarded `git ls-remote` is what aborted onboarding), and the run still
// finishes by persisting the configuration last.
func TestInitApplySkipsPublicationWithoutOrigin(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	draft.generateAgent = true
	ops, calls := recordingInitOperations()
	ops.hasOriginRemote = func() bool { return false }

	var err error
	output := captureInitStdout(t, func() { err = applyInitDraft(draft, ops) })
	if err != nil {
		t.Fatalf("applyInitDraft() error = %v, want a run that completes without origin", err)
	}

	want := []string{
		"validate:main", "validate:develop", "validate:uat",
		"ensure:main", "ensure:develop", "ensure:uat",
		"agent", "save",
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("calls = %v, want no remote/push operations and config saved last %v", *calls, want)
	}
	if got := strings.Count(output, initPublicationSkipWording); got != 1 {
		t.Fatalf("skip report rendered %d times, want exactly one line %q in:\n%s", got, initPublicationSkipWording, output)
	}
}

// TestInitApplyTreatsMissingProbesAsAnAvailableRepository keeps the nil seam the
// file already uses for generateAgentFiles: an operation that is not wired must
// not fabricate a failure, or a partially built initOperations would block a
// legitimate run.
func TestInitApplyTreatsMissingProbesAsAnAvailableRepository(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	ops, calls := recordingInitOperations()
	ops.hasCommits = nil
	ops.hasOriginRemote = nil

	if err := applyInitDraft(draft, ops); err != nil {
		t.Fatalf("applyInitDraft() error = %v, want unset probes to stay permissive", err)
	}
	if !containsCall(*calls, "push:main") || !containsCall(*calls, "save") {
		t.Fatalf("unset probes changed the run: %v", *calls)
	}
}

func TestInitApplyRejectsInvalidPrimaryBranchBeforeMutation(t *testing.T) {
	draft := validInitDraft()
	draft.config.Branches.Develop = "bad name"
	draft.branches = []string{"main", "bad name", "uat"}
	ops, calls := recordingInitOperations()
	ops.validateBranch = func(branch string) error {
		*calls = append(*calls, "validate:"+branch)
		if branch == "bad name" {
			return errors.New("fatal: 'bad name' is not a valid branch name")
		}
		return nil
	}

	err := applyInitDraft(draft, ops)
	if err == nil || !strings.Contains(err.Error(), `invalid development branch "bad name"`) {
		t.Fatalf("applyInitDraft() error = %v, want invalid development branch", err)
	}
	assertNoInitMutations(t, *calls)
}

func TestInitApplyValidatesConfigBeforeMutation(t *testing.T) {
	draft := validInitDraft()
	draft.config.Branches.Releases = draft.config.Branches.Features
	ops, calls := recordingInitOperations()

	err := applyInitDraft(draft, ops)
	if err == nil || !strings.Contains(err.Error(), "validate onboarding configuration") {
		t.Fatalf("applyInitDraft() error = %v, want configuration validation error", err)
	}
	assertNoInitMutations(t, *calls)
}

func TestInitApplyValidatesAllPrimaryRefsBeforeMutation(t *testing.T) {
	draft := validInitDraft()
	ops, calls := recordingInitOperations()

	if err := applyInitDraft(draft, ops); err != nil {
		t.Fatalf("applyInitDraft() error = %v", err)
	}
	wantPrefix := []string{"validate:main", "validate:develop", "validate:uat", "ensure:main"}
	if got := (*calls)[:len(wantPrefix)]; !reflect.DeepEqual(got, wantPrefix) {
		t.Fatalf("initial calls = %v, want validation before first mutation %v", got, wantPrefix)
	}
}

func TestInitApplyUsesCollectedAnswersAndWritesConfigLast(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	draft.generateAgent = true
	ops, calls := recordingInitOperations()

	if err := applyInitDraft(draft, ops); err != nil {
		t.Fatalf("applyInitDraft() error = %v", err)
	}
	want := []string{
		"validate:main", "validate:develop", "validate:uat",
		"ensure:main", "ensure:develop", "ensure:uat",
		"remote:main", "push:main", "remote:develop", "push:develop", "remote:uat", "push:uat",
		"agent", "save",
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("calls = %v, want final-only config persistence sequence %v", *calls, want)
	}
}

func TestInitApplyStopsAtBranchCreationFailure(t *testing.T) {
	draft := validInitDraft()
	ops, calls := recordingInitOperations()
	ops.ensureBranch = func(branch string) error {
		*calls = append(*calls, "ensure:"+branch)
		if branch == "develop" {
			return errors.New("fatal: cannot lock ref")
		}
		return nil
	}

	err := applyInitDraft(draft, ops)
	assertInitFailure(t, err, "create or reuse local branch \"develop\"", "fatal: cannot lock ref", "Local branches retained: main", "No remote branches were published", "rerun `dflow init`")
	if got, want := *calls, []string{"validate:main", "validate:develop", "validate:uat", "ensure:main", "ensure:develop"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %v, want fail-fast sequence %v", got, want)
	}
}

func TestInitApplyReportsPushFailureBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		failed     string
		remoteDone string
		wantCalls  []string
	}{
		{name: "second push", failed: "develop", remoteDone: "main", wantCalls: []string{"remote:main", "push:main", "remote:develop", "push:develop"}},
		{name: "third push", failed: "uat", remoteDone: "main, develop", wantCalls: []string{"remote:main", "push:main", "remote:develop", "push:develop", "remote:uat", "push:uat"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			draft := validInitDraft()
			draft.push = true
			ops, calls := recordingInitOperations()
			ops.pushBranch = func(branch string) error {
				*calls = append(*calls, "push:"+branch)
				if branch == tc.failed {
					return errors.New("fatal: remote rejected onboarding ref")
				}
				return nil
			}

			err := applyInitDraft(draft, ops)
			assertInitFailure(t, err, "publish branch \""+tc.failed+"\"", "fatal: remote rejected onboarding ref", "Local branches retained: main, develop, uat", "Remote branches retained: "+tc.remoteDone, "rerun `dflow init`")
			var pushCalls []string
			for _, call := range *calls {
				if strings.HasPrefix(call, "remote:") || strings.HasPrefix(call, "push:") {
					pushCalls = append(pushCalls, call)
				}
			}
			if !reflect.DeepEqual(pushCalls, tc.wantCalls) {
				t.Fatalf("push calls = %v, want %v", pushCalls, tc.wantCalls)
			}
			if containsCall(*calls, "save") {
				t.Fatalf("configuration was written after failed push: %v", *calls)
			}
		})
	}
}

func TestInitApplyForceFailureUsesForceRetryCommand(t *testing.T) {
	draft := validInitDraft()
	draft.force = true
	ops, calls := recordingInitOperations()
	ops.ensureBranch = func(string) error { return errors.New("fatal: cannot create branch") }

	err := applyInitDraft(draft, ops)
	assertInitFailure(t, err, "rerun `dflow init --force`")
	if strings.Contains(err.Error(), "rerun `dflow init`") || containsCall(*calls, "save") {
		t.Fatalf("force failure guidance or persistence is unsafe: error=%v calls=%v", err, *calls)
	}
}

func TestInitApplyRetryReusesPublishedRefsAndWritesConfigLast(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	ops, calls := recordingInitOperations()
	ops.inspectRemoteBranch = func(branch string) (initRemoteBranch, error) {
		*calls = append(*calls, "remote:"+branch)
		if branch == "uat" {
			return initRemoteBranch{localRevision: "uat-revision"}, nil
		}
		return initRemoteBranch{exists: true, localRevision: branch + "-revision", remoteRevision: branch + "-revision"}, nil
	}

	if err := applyInitDraft(draft, ops); err != nil {
		t.Fatalf("applyInitDraft() error = %v", err)
	}
	if containsCall(*calls, "push:main") || containsCall(*calls, "push:develop") {
		t.Fatalf("retry republished existing remote refs: %v", *calls)
	}
	if got, want := (*calls)[len(*calls)-2:], []string{"push:uat", "save"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("final calls = %v, want %v (config saved last)", got, want)
	}
}

func TestInitApplyRefusesMismatchedExistingRemote(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	ops, calls := recordingInitOperations()
	ops.inspectRemoteBranch = func(branch string) (initRemoteBranch, error) {
		*calls = append(*calls, "remote:"+branch)
		return initRemoteBranch{exists: true, localRevision: "local-abc", remoteRevision: "remote-def"}, nil
	}

	err := applyInitDraft(draft, ops)
	assertInitFailure(t, err, "verify existing remote branch \"main\"", "local-abc", "remote-def", "refusing to overwrite", "Remote branches retained: main", "rerun `dflow init`")
	if containsCall(*calls, "push:main") || containsCall(*calls, "save") {
		t.Fatalf("mismatched remote was overwritten or config was saved: %v", *calls)
	}
}

func TestInitApplyAgentFailureLeavesConfigUnchanged(t *testing.T) {
	draft := validInitDraft()
	draft.generateAgent = true
	ops, calls := recordingInitOperations()
	ops.generateAgentFiles = func(*flow.Config) error {
		*calls = append(*calls, "agent")
		return errors.New("permission denied")
	}

	err := applyInitDraft(draft, ops)
	assertInitFailure(t, err, "generate agent workflow", "permission denied", ".dflow.yaml was not changed")
	if containsCall(*calls, "save") {
		t.Fatalf("configuration was written after agent generation failure: %v", *calls)
	}
}

func TestInitApplyRemoteLookupFailurePreservesDiagnostic(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	ops, calls := recordingInitOperations()
	ops.inspectRemoteBranch = func(branch string) (initRemoteBranch, error) {
		*calls = append(*calls, "remote:"+branch)
		return initRemoteBranch{}, errors.New("fatal: unable to access origin")
	}

	err := applyInitDraft(draft, ops)
	assertInitFailure(t, err, "check remote branch \"main\"", "fatal: unable to access origin", "Local branches retained: main, develop, uat", "No remote branches were published")
	if containsCall(*calls, "save") {
		t.Fatalf("configuration was written after remote lookup failure: %v", *calls)
	}
}

func TestInitRemoteBranchInspectionUsesDflowCWD(t *testing.T) {
	if testing.Short() {
		t.Skip("requires git")
	}
	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	local := filepath.Join(root, "local")
	decoy := filepath.Join(root, "decoy")
	runInitGit(t, root, "init", "--bare", origin)
	runInitGit(t, root, "init", local)
	runInitGit(t, local, "config", "user.email", "init-test@example.com")
	runInitGit(t, local, "config", "user.name", "Init Test")
	if err := os.WriteFile(filepath.Join(local, "README.md"), []byte("target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runInitGit(t, local, "add", "README.md")
	runInitGit(t, local, "commit", "-m", "target")
	runInitGit(t, local, "branch", "-M", "main")
	runInitGit(t, local, "remote", "add", "origin", origin)
	runInitGit(t, local, "push", "-u", "origin", "main")
	targetRevision := runInitGitOutput(t, local, "rev-parse", "refs/heads/main")

	runInitGit(t, root, "init", decoy)
	runInitGit(t, decoy, "config", "user.email", "init-test@example.com")
	runInitGit(t, decoy, "config", "user.name", "Init Test")
	if err := os.WriteFile(filepath.Join(decoy, "README.md"), []byte("decoy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runInitGit(t, decoy, "add", "README.md")
	runInitGit(t, decoy, "commit", "-m", "decoy")
	runInitGit(t, decoy, "branch", "-M", "main")
	decoyRevision := runInitGitOutput(t, decoy, "rev-parse", "refs/heads/main")
	if targetRevision == decoyRevision {
		t.Fatal("fixture revisions unexpectedly match")
	}

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(decoy); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	t.Setenv("DFLOW_CWD", local)

	state, err := inspectInitRemoteBranch("main")
	if err != nil {
		t.Fatalf("inspectInitRemoteBranch() error = %v", err)
	}
	if !state.exists || state.localRevision != targetRevision || state.remoteRevision != targetRevision {
		t.Fatalf("state = %+v, want target revision %q for local and origin", state, targetRevision)
	}
}

func TestInitBranchValidationUsesGitRefRules(t *testing.T) {
	for _, tc := range []struct {
		name    string
		branch  string
		wantErr bool
	}{
		{name: "ordinary branch", branch: "release/1.2"},
		{name: "contains space", branch: "bad name", wantErr: true},
		{name: "starts with dash", branch: "-bad", wantErr: true},
		{name: "contains dot sequence", branch: "bad..name", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInitBranch(tc.branch)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateInitBranch(%q) error = %v, wantErr %v", tc.branch, err, tc.wantErr)
			}
		})
	}
}

func validInitDraft() initDraft {
	cfg := flow.Config{}
	cfg.Branches.Main, cfg.Branches.Develop, cfg.Branches.Uat = "main", "develop", "uat"
	cfg.Branches.Features, cfg.Branches.Releases = "feature/", "release/"
	cfg.Branches.Hotfixes, cfg.Branches.Bugfixes = "hotfix/", "bugfix/"
	cfg.Flow.Feature = flow.BranchFlowRule{Base: "develop", FinishTargets: []string{"develop", "uat"}}
	cfg.Flow.Release = flow.BranchFlowRule{Base: "uat", FinishTargets: []string{"main", "develop"}}
	cfg.Flow.Hotfix = flow.BranchFlowRule{Base: "main", FinishTargets: []string{"main", "develop", "uat"}}
	cfg.Flow.Bugfix = flow.BranchFlowRule{Base: "uat", FinishTargets: []string{"uat", "develop"}}
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
		"main": {MergeMode: "manual"}, "develop": {MergeMode: "manual"}, "uat": {MergeMode: "manual"},
	}
	return initDraft{config: cfg, branches: []string{"main", "develop", "uat"}}
}

func recordingInitOperations() (initOperations, *[]string) {
	calls := []string{}
	return initOperations{
		// The repository-state probes default to the healthy case and record
		// nothing, so every existing sequence assertion keeps describing the
		// onboarding steps rather than the preflights in front of them. Tests that
		// need the other answer replace one of these two.
		hasCommits:      func() bool { return true },
		hasOriginRemote: func() bool { return true },
		validateBranch:  func(branch string) error { calls = append(calls, "validate:"+branch); return nil },
		ensureBranch:    func(branch string) error { calls = append(calls, "ensure:"+branch); return nil },
		inspectRemoteBranch: func(branch string) (initRemoteBranch, error) {
			calls = append(calls, "remote:"+branch)
			return initRemoteBranch{localRevision: branch + "-revision"}, nil
		},
		pushBranch: func(branch string) error { calls = append(calls, "push:"+branch); return nil },
		generateAgentFiles: func(*flow.Config) error {
			calls = append(calls, "agent")
			return nil
		},
		saveConfig: func(*flow.Config) error { calls = append(calls, "save"); return nil },
	}, &calls
}

// captureInitStdout runs fn with os.Stdout redirected to a pipe and returns what
// it wrote. The onboarding progress lines go through the shared output helpers,
// which print to stdout, so redirection is the only way to observe them in a
// plain unit test without a pseudo-terminal.
func captureInitStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout capture pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = original })

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout capture writer: %v", err)
	}
	captured, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close stdout capture reader: %v", err)
	}
	return string(captured)
}

func assertNoInitMutations(t *testing.T, calls []string) {
	t.Helper()
	for _, call := range calls {
		if !strings.HasPrefix(call, "validate:") {
			t.Fatalf("invalid draft mutated state through calls %v", calls)
		}
	}
}

func assertInitFailure(t *testing.T, err error, wants ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("applyInitDraft() error = nil, want failure")
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}

func runInitGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	runInitGitOutput(t, dir, args...)
}

func runInitGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
