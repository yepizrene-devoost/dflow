package tests

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yepizrene-devoost/dflow/pkg/agent"
	"github.com/yepizrene-devoost/dflow/pkg/flow"
	"gopkg.in/yaml.v3"
)

// This file pins the discovery wiring of `dflow agent` and `dflow init` against
// the real binary: which instruction files the commands bring the workflow
// reference to, which ones they deliberately leave alone, that a bad --agents
// value writes nothing at all, and that --json is a strictly read-only render.
//
// The assertions are exact bytes rather than substrings wherever the bytes are
// the contract: the document is compared to the rendered workflow, and every
// instruction file is compared to the reference block it must carry (which also
// proves there is exactly one block, since a whole-file comparison cannot pass
// with a second copy in it).

// agentWorkflowDefaultPath is the document path `dflow agent` writes when --path
// is not given, and therefore the path each reference block must name.
const agentWorkflowDefaultPath = ".agents/workflows/dflow.md"

// agentCLIConfig is a hand-written .dflow.yaml with recognizable values. It is
// written directly to disk so the document the commands render is fully
// controlled by this file and not by an in-process SaveConfig round trip.
const agentCLIConfig = `branches:
  main: main
  develop: develop
  uat: uat
  features: "feature/"
  releases: "release/"
  hotfixes: "hotfix/"
  bugfixes: "bugfix/"
flow:
  feature:
    base: develop
    finish_targets:
      - develop
      - uat
workflow:
  default_merge_mode: manual
  branch_rules:
    develop:
      merge_mode: auto
`

// expectedReferenceBlock is the exact block every instruction file must carry
// for the document path a run actually wrote.
//
// It is spelled out here, markers and all, instead of being computed from
// pkg/agent: this is the end-to-end pin of the user-visible bytes, and a
// comparison against the library that produced them could not see a rendered
// block change.
func expectedReferenceBlock(workflowPath string) string {
	return "<!-- dflow:workflow-reference -->\n" +
		"## dflow Workflow\n" +
		"Read `" + workflowPath + "` for branch types, merge rules, and finish flow.\n" +
		"<!-- /dflow:workflow-reference -->\n"
}

// newAgentRepo creates a git repository carrying the hand-written config above,
// which is the environment both `dflow agent` and `dflow init` need.
func newAgentRepo(t *testing.T) string {
	t.Helper()

	repo := initTempGitRepo(t)
	writeRepoFile(t, filepath.Join(repo, ".dflow.yaml"), agentCLIConfig)
	return repo
}

// expectedAgentDoc renders the workflow document the commands must write for the
// .dflow.yaml currently in repo. Reading the file back and rendering from the
// parsed configuration is what makes the document assertion a real one: the
// bytes have to describe the configuration on disk.
func expectedAgentDoc(t *testing.T, repo string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(repo, ".dflow.yaml"))
	if err != nil {
		t.Fatalf("failed to read the generated .dflow.yaml: %v", err)
	}
	var cfg flow.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse the generated .dflow.yaml: %v", err)
	}
	return string(agent.GenerateAgentDoc(&cfg))
}

// writeRepoFile writes content to path, failing the test on error.
func writeRepoFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// requireFileContent asserts path exists and holds exactly want.
func requireFileContent(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if got := string(data); got != want {
		t.Fatalf("%s does not hold the expected bytes:\n--- want ---\n%s\n--- got ---\n%s", filepath.Base(path), want, got)
	}
}

// requireFileAbsent asserts path does not exist, whatever the error flavour.
func requireFileAbsent(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("%s exists, want it untouched", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

// requireAbsPathUnder asserts got is the absolute form of rel under root.
//
// A child process reports the physical working directory (on macOS
// /private/var/...), while the test holds the symlinked form t.TempDir handed
// it, so both spellings are accepted: the relative tail under the repository is
// what the contract actually pins.
func requireAbsPathUnder(t *testing.T, got, root, rel string) {
	t.Helper()

	want := filepath.Join(root, rel)
	if got == want {
		return
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil && got == filepath.Join(resolved, rel) {
		return
	}
	t.Fatalf("path = %q, want %q (or its symlink-resolved form)", got, want)
}

// TestAgentCLIWritesDocumentAndAgentsReference pins the default run: the
// document is written and AGENTS.md is created carrying exactly the reference
// block that names it.
func TestAgentCLIWritesDocumentAndAgentsReference(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent")
	if exitCode != 0 {
		t.Fatalf("dflow agent exited %d, want 0\n%s", exitCode, output)
	}

	requireFileContent(t, filepath.Join(repo, agentWorkflowDefaultPath), expectedAgentDoc(t, repo))
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
}

// TestAgentCLIReferenceStaysSingleOnForcedRerun pins idempotence: a --force run
// rewrites the document but must not append a second reference block.
func TestAgentCLIReferenceStaysSingleOnForcedRerun(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	if output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent"); exitCode != 0 {
		t.Fatalf("first dflow agent exited %d, want 0\n%s", exitCode, output)
	}
	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--force")
	if exitCode != 0 {
		t.Fatalf("second dflow agent --force exited %d, want 0\n%s", exitCode, output)
	}

	// A whole-file comparison against one block is exactly the "one block, no
	// duplicates, no leftovers" assertion.
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
}

// TestAgentCLIUpdatesExistingClaudeMd pins the refresh half of maintainer
// decision 2: a project that already uses Claude Code gets CLAUDE.md kept
// current, after its existing bytes.
func TestAgentCLIUpdatesExistingClaudeMd(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	existing := "# Claude project instructions\n"
	claudePath := filepath.Join(repo, "CLAUDE.md")
	writeRepoFile(t, claudePath, existing)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent")
	if exitCode != 0 {
		t.Fatalf("dflow agent exited %d, want 0\n%s", exitCode, output)
	}

	requireFileContent(t, claudePath, existing+"\n"+expectedReferenceBlock(agentWorkflowDefaultPath))
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
}

// TestAgentCLIDoesNotCreateClaudeMd pins the create half of maintainer decision
// 2: without a CLAUDE.md, a default run must not drop a surprise one into the
// repository.
func TestAgentCLIDoesNotCreateClaudeMd(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent")
	if exitCode != 0 {
		t.Fatalf("dflow agent exited %d, want 0\n%s", exitCode, output)
	}

	requireFileAbsent(t, filepath.Join(repo, "CLAUDE.md"))
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
}

// TestAgentCLINamedClaudeCreatesClaudeMdOnly pins the explicit selection: naming
// claude authorizes creating CLAUDE.md, and naming only claude leaves AGENTS.md
// alone because no selected agent reads it.
func TestAgentCLINamedClaudeCreatesClaudeMdOnly(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--agents", "claude")
	if exitCode != 0 {
		t.Fatalf("dflow agent --agents claude exited %d, want 0\n%s", exitCode, output)
	}

	requireFileContent(t, filepath.Join(repo, "CLAUDE.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
	requireFileAbsent(t, filepath.Join(repo, "AGENTS.md"))
}

// TestAgentCLIAgentSpecSelection triangulates the named selection on the three
// edges around the claude case: "all" names claude, so CLAUDE.md is created
// alongside AGENTS.md; "pi" does not, even though pi also reads CLAUDE.md — only
// a named agent's PRIMARY instruction file is created; and an empty spec is auto,
// not a named "all", so it must not create a CLAUDE.md either.
func TestAgentCLIAgentSpecSelection(t *testing.T) {
	cases := []struct {
		name       string
		spec       string
		wantAgents bool
		wantClaude bool
	}{
		{name: "all", spec: "all", wantAgents: true, wantClaude: true},
		{name: "pi", spec: "pi", wantAgents: true, wantClaude: false},
		{name: "empty spec is auto", spec: "", wantAgents: true, wantClaude: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setUpCLIEnv(t)
			binary := buildDflowCLI(t)
			repo := newAgentRepo(t)

			output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--agents", tc.spec)
			if exitCode != 0 {
				t.Fatalf("dflow agent --agents %q exited %d, want 0\n%s", tc.spec, exitCode, output)
			}

			references := []struct {
				file string
				want bool
			}{
				{file: "AGENTS.md", want: tc.wantAgents},
				{file: "CLAUDE.md", want: tc.wantClaude},
			}
			for _, ref := range references {
				path := filepath.Join(repo, ref.file)
				if !ref.want {
					requireFileAbsent(t, path)
					continue
				}
				requireFileContent(t, path, expectedReferenceBlock(agentWorkflowDefaultPath))
			}
		})
	}
}

// TestAgentCLIUnknownAgentWritesNothing pins the ordering guarantee: an unknown
// agent id fails the run and leaves no document, no directory and no reference
// behind, because the selection is validated before anything is written.
func TestAgentCLIUnknownAgentWritesNothing(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--agents", "bogus")
	if exitCode == 0 {
		t.Fatalf("dflow agent --agents bogus exited 0, want non-zero\n%s", output)
	}
	if !strings.Contains(output, "bogus") {
		t.Fatalf("the failure must name the unknown agent id, got:\n%s", output)
	}

	requireFileAbsent(t, filepath.Join(repo, agentWorkflowDefaultPath))
	requireFileAbsent(t, filepath.Join(repo, ".agents"))
	requireFileAbsent(t, filepath.Join(repo, "AGENTS.md"))
	requireFileAbsent(t, filepath.Join(repo, "CLAUDE.md"))
}

// TestAgentCLIJSONReportsThePlanAndWritesNothing pins --json as a strictly
// read-only render: one JSON document and nothing else, the byte-compatible
// path/content keys, the reference plan, and no file on disk at all.
func TestAgentCLIJSONReportsThePlanAndWritesNothing(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--json")
	if exitCode != 0 {
		t.Fatalf("dflow agent --json exited %d, want 0\n%s", exitCode, output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	if len(doc) != 4 {
		t.Fatalf("the document carries %d top-level keys, want exactly path, content, references and skills: %v", len(doc), doc)
	}
	path, ok := doc["path"].(string)
	if !ok {
		t.Fatalf("path = %#v, want a string", doc["path"])
	}
	requireAbsPathUnder(t, path, repo, agentWorkflowDefaultPath)
	requireJSONString(t, doc, "content", expectedAgentDoc(t, repo))

	references := requireJSONArray(t, doc, "references")
	if len(references) != 1 {
		t.Fatalf("references has %d entries, want 1 (AGENTS.md):\n%v", len(references), references)
	}
	entry, ok := references[0].(map[string]any)
	if !ok {
		t.Fatalf("references[0] = %#v, want an object", references[0])
	}
	if len(entry) != 3 {
		t.Fatalf("references[0] carries %d keys, want exactly file, agents and create: %v", len(entry), entry)
	}
	requireJSONString(t, entry, "file", "AGENTS.md")
	requireJSONStringSlice(t, entry, "agents", []string{"pi", "codex", "opencode"})
	requireJSONBool(t, entry, "create", true)

	// The skills plan is reported without --install too: it is the plan the flag
	// would apply, and a read-only render of it costs nothing. Auto mode targets
	// the portable root only, and an uninstalled skill is reported as absent.
	skills := requireJSONArray(t, doc, "skills")
	if len(skills) != 1 {
		t.Fatalf("skills has %d entries, want 1 (the portable root):\n%v", len(skills), skills)
	}
	skillEntry, ok := skills[0].(map[string]any)
	if !ok {
		t.Fatalf("skills[0] = %#v, want an object", skills[0])
	}
	if len(skillEntry) != 4 {
		t.Fatalf("skills[0] carries %d keys, want exactly root, dir, agents and exists: %v", len(skillEntry), skillEntry)
	}
	requireJSONString(t, skillEntry, "root", "~/.agents/skills")
	requireJSONString(t, skillEntry, "dir", filepath.Join("~/.agents/skills", agent.SkillName))
	requireJSONStringSlice(t, skillEntry, "agents", []string{"pi"})
	requireJSONBool(t, skillEntry, "exists", false)

	requireFileAbsent(t, filepath.Join(repo, agentWorkflowDefaultPath))
	requireFileAbsent(t, filepath.Join(repo, ".agents"))
	requireFileAbsent(t, filepath.Join(repo, "AGENTS.md"))
	requireFileAbsent(t, filepath.Join(home, ".agents"))
}

// TestAgentCLIJSONRendersWhenDocumentExists pins the read-only render against
// the document it renders: an existing document must not make --json fail with
// "already exists", and the bytes already on disk must survive untouched.
func TestAgentCLIJSONRendersWhenDocumentExists(t *testing.T) {
	// HOME is redirected even though no HOME path is asserted here: the report
	// now carries the skill plan, whose existence check reads the user's skills
	// directory, and a pin must not read the machine's own installed skills.
	skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	docPath := filepath.Join(repo, agentWorkflowDefaultPath)
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatalf("failed to create the document directory: %v", err)
	}
	sentinel := "# hand-written document that --json must not touch\n"
	writeRepoFile(t, docPath, sentinel)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--json")
	if exitCode != 0 {
		t.Fatalf("dflow agent --json over an existing document exited %d, want 0\n%s", exitCode, output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	requireJSONString(t, doc, "content", expectedAgentDoc(t, repo))
	requireFileContent(t, docPath, sentinel)
	requireFileAbsent(t, filepath.Join(repo, "AGENTS.md"))
}

// realGOMODCACHE is the host's module cache, captured at package initialization,
// before any test can redirect HOME.
//
// Redirecting HOME for a skill pin would otherwise change how the `go build` in
// the shared CLI helper resolves its module cache. With no GOMODCACHE set, that
// cache is `$GOPATH/pkg/mod` under the process's HOME, so the build would download
// modules into the throwaway home — and the module cache is deliberately
// read-only, which makes t.TempDir's own RemoveAll cleanup fail and turns an
// otherwise green pin into a cleanup error. Anchoring the variable keeps the build
// identical to the one the host would run on its own and keeps the temporary home
// free of build state. It mirrors realGOCACHE in clibuild_test.go.
//
// Package-level initialization runs before TestMain and before any t.Setenv, so
// the value captured is the host's real cache. An empty value leaves the child's
// environment unchanged: correct, only slower.
var realGOMODCACHE = func() string {
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}()

// skillInstallEnv extends setUpCLIEnv with a redirected HOME, and returns it.
//
// Installing a skill resolves the "~"-prefixed registry roots through the user's
// home directory, so an install-capable pin that left HOME alone would write into
// the developer's real ~/.agents/skills or ~/.claude/skills — and a --json pin
// would read them, making its report depend on the machine it runs on. setUpCLIEnv
// already keeps Git and the update-check cache off the real home; this keeps
// skills off it as well, and pins the module cache so the shared build does not
// move into the temporary home with it.
func skillInstallEnv(t *testing.T) string {
	t.Helper()

	setUpCLIEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if realGOMODCACHE != "" {
		t.Setenv("GOMODCACHE", realGOMODCACHE)
	}
	return home
}

// skillPath is the installed skill file under a skills root.
func skillPath(root string) string {
	return filepath.Join(root, agent.SkillName, agent.SkillFileName)
}

// userSkillPath is the installed skill file inside the redirected home for one
// agent directory, such as ".agents" or ".claude".
func userSkillPath(home, dir string) string {
	return skillPath(filepath.Join(home, dir, "skills"))
}

// projectSkillPath is the installed skill file inside the repository for one
// agent directory.
func projectSkillPath(repo, dir string) string {
	return skillPath(filepath.Join(repo, dir, "skills"))
}

// requireSingleSkillFile asserts a skills directory holds exactly the skill file.
// It is how a temporary file left behind by a non-atomic write is caught.
func requireSingleSkillFile(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != 1 || names[0] != agent.SkillFileName {
		t.Fatalf("%s holds %v, want exactly [%s]", dir, names, agent.SkillFileName)
	}
}

// The default install writes the PORTABLE root only — the one file pi, codex and
// opencode all discover — and leaves every private agent directory in $HOME
// alone. An unattended run must not scatter four copies through the user's home,
// so the private roots are opt-in through --agents.
func TestAgentCLIInstallWritesPortableSkillRootOnly(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install exited %d, want 0\n%s", exitCode, output)
	}

	doc := expectedAgentDoc(t, repo)
	installed := userSkillPath(home, ".agents")
	requireFileContent(t, installed, doc)
	if !strings.Contains(output, installed) {
		t.Errorf("the run must name the skill file it installed (%s), got:\n%s", installed, output)
	}

	requireFileAbsent(t, filepath.Join(home, ".claude"))
	requireFileAbsent(t, filepath.Join(home, ".codex"))
	requireFileAbsent(t, filepath.Join(home, ".config"))

	// --install adds to what the command already does: the document and its
	// AGENTS.md reference are still written exactly as before.
	requireFileContent(t, filepath.Join(repo, agentWorkflowDefaultPath), doc)
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
}

// Naming claude installs claude's own root in $HOME — and nothing else in either
// direction: no portable root, and no AGENTS.md, because no selected agent reads
// it.
func TestAgentCLIInstallNamedClaudeWritesOnlyClaudeRoot(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--agents", "claude")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install --agents claude exited %d, want 0\n%s", exitCode, output)
	}

	doc := expectedAgentDoc(t, repo)
	requireFileContent(t, userSkillPath(home, ".claude"), doc)
	requireFileAbsent(t, filepath.Join(home, ".agents"))
	requireFileAbsent(t, filepath.Join(home, ".codex"))
	requireFileAbsent(t, filepath.Join(home, ".config"))

	requireFileContent(t, filepath.Join(repo, "CLAUDE.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
	requireFileAbsent(t, filepath.Join(repo, "AGENTS.md"))
}

// Naming every agent widens the install to all four user-scope roots, including
// the private ones a plain run leaves alone.
func TestAgentCLIInstallAllWritesEveryUserRoot(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--agents", "all")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install --agents all exited %d, want 0\n%s", exitCode, output)
	}

	doc := expectedAgentDoc(t, repo)
	for _, dir := range []string{".agents", ".codex", filepath.Join(".config", "opencode"), ".claude"} {
		requireFileContent(t, userSkillPath(home, dir), doc)
	}
}

// --install --local keeps the skill inside the repository: the project's own
// portable root, and nothing in the user's home.
func TestAgentCLIInstallLocalWritesProjectSkill(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--local")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install --local exited %d, want 0\n%s", exitCode, output)
	}

	doc := expectedAgentDoc(t, repo)
	requireFileContent(t, projectSkillPath(repo, ".agents"), doc)
	requireFileAbsent(t, filepath.Join(repo, ".claude"))
	requireFileAbsent(t, filepath.Join(home, ".agents"))
	requireFileAbsent(t, filepath.Join(home, ".claude"))
}

// --agents all in the LOCAL scope writes both project roots, which is the pair
// the portable root does not cover for a repository that uses Claude Code.
func TestAgentCLIInstallLocalAllWritesBothProjectRoots(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--local", "--agents", "all")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install --local --agents all exited %d, want 0\n%s", exitCode, output)
	}

	doc := expectedAgentDoc(t, repo)
	requireFileContent(t, projectSkillPath(repo, ".agents"), doc)
	requireFileContent(t, projectSkillPath(repo, ".claude"), doc)
	requireFileAbsent(t, filepath.Join(home, ".agents"))
	requireFileAbsent(t, filepath.Join(home, ".claude"))
}

// A second --install is idempotent: one file, identical bytes, no temporary
// leftovers. It also must NOT require --force: the document is protected and the
// skill is generated, so an existing document is no reason to push the user onto
// the one destructive path in the command. The pin writes a distinctive sentinel
// to the document before the second run — standing in for a team's hand edit —
// and asserts the document survives byte for byte while the skill is refreshed
// from the freshly generated bytes and the references are rewired.
func TestAgentCLIInstallIsIdempotent(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	if output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install"); exitCode != 0 {
		t.Fatalf("first dflow agent --install exited %d, want 0\n%s", exitCode, output)
	}
	doc := expectedAgentDoc(t, repo)
	installed := userSkillPath(home, ".agents")
	requireFileContent(t, installed, doc)

	// The document a team hand-edited: bytes this install must never overwrite.
	sentinel := "# hand-edited workflow document the install must protect\n"
	docPath := filepath.Join(repo, agentWorkflowDefaultPath)
	writeRepoFile(t, docPath, sentinel)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install")
	if exitCode != 0 {
		t.Fatalf("the second dflow agent --install exited %d, want 0: an existing document must not force --force on an install\n%s", exitCode, output)
	}
	// The document is protected, so it keeps its exact bytes...
	requireFileContent(t, docPath, sentinel)
	// ...the skill is generated from the freshly rendered document...
	requireFileContent(t, installed, doc)
	// ...the reference is still wired...
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
	// ...and the run must not claim a document it never generated.
	if strings.Contains(output, "Generated agent workflow") {
		t.Fatalf("the run claimed to generate the document it left untouched:\n%s", output)
	}
	requireSingleSkillFile(t, filepath.Dir(installed))

	if output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--force"); exitCode != 0 {
		t.Fatalf("dflow agent --install --force exited %d, want 0\n%s", exitCode, output)
	}
	requireFileContent(t, docPath, doc)
	requireFileContent(t, installed, doc)
	requireSingleSkillFile(t, filepath.Dir(installed))
}

// TestAgentCLIInstallProtectsHandEditedDocument is the scenario that motivates
// --install not requiring --force: a repository whose workflow document was
// hand-edited, on a FIRST install. The install must succeed, because the document
// is protected and the skill is generated — the two artifacts are allowed to
// differ — while driving the user to the destructive --force to install a skill
// would be exactly backwards.
func TestAgentCLIInstallProtectsHandEditedDocument(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	docPath := filepath.Join(repo, agentWorkflowDefaultPath)
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatalf("failed to create the document directory: %v", err)
	}
	sentinel := "# hand-edited workflow document a project owns\n\nOnly --force may replace me.\n"
	writeRepoFile(t, docPath, sentinel)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install over a hand-edited document exited %d, want 0; installing must not require --force\n%s", exitCode, output)
	}

	// The document keeps its exact bytes...
	requireFileContent(t, docPath, sentinel)
	// ...the skill is the generated one...
	doc := expectedAgentDoc(t, repo)
	requireFileContent(t, userSkillPath(home, ".agents"), doc)
	// ...the reference is still wired...
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
	// ...and the run never claims to have generated the document.
	if strings.Contains(output, "Generated agent workflow") {
		t.Fatalf("the run claimed to generate the document it protected:\n%s", output)
	}
}

// TestAgentCLIWithoutInstallRefusesExistingDocument pins the other half of the
// contract: without --install the document guard is unchanged. An existing
// document and no --force is still the same refusal as before, and nothing at
// all is written.
func TestAgentCLIWithoutInstallRefusesExistingDocument(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	docPath := filepath.Join(repo, agentWorkflowDefaultPath)
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatalf("failed to create the document directory: %v", err)
	}
	sentinel := "# hand-edited workflow document the plain guard must refuse to overwrite\n"
	writeRepoFile(t, docPath, sentinel)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent")
	if exitCode == 0 {
		t.Fatalf("dflow agent over an existing document exited 0, want the guard to refuse it\n%s", output)
	}
	if !strings.Contains(output, "--force") {
		t.Fatalf("the refusal must name the way out (--force), got:\n%s", output)
	}
	requireFileContent(t, docPath, sentinel)
	requireFileAbsent(t, filepath.Join(repo, "AGENTS.md"))
}

// TestAgentCLIForceRegeneratesDocument keeps the destructive path honest: --force
// is the one run allowed to replace a hand-edited document, with or without
// --install.
func TestAgentCLIForceRegeneratesDocument(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	docPath := filepath.Join(repo, agentWorkflowDefaultPath)
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatalf("failed to create the document directory: %v", err)
	}
	sentinel := "# hand-edited workflow document --force is expected to replace\n"

	for _, args := range [][]string{
		{"agent", "--force"},
		{"agent", "--install", "--force"},
	} {
		writeRepoFile(t, docPath, sentinel)
		output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, args...)
		if exitCode != 0 {
			t.Fatalf("dflow %v exited %d, want 0\n%s", args, exitCode, output)
		}
		requireFileContent(t, docPath, expectedAgentDoc(t, repo))
		if !strings.Contains(output, "Generated agent workflow") {
			t.Fatalf("dflow %v must report the regenerated document, got:\n%s", args, output)
		}
	}

	// The --install --force run installs the skill from the same bytes it wrote.
	requireFileContent(t, userSkillPath(home, ".agents"), expectedAgentDoc(t, repo))
}

// Reinstalling over a stale skill file needs no --force: those bytes are dflow's
// own artifact, so refreshing them is the point rather than an override. The
// document is removed first because its own guard is a separate decision that
// does require --force.
func TestAgentCLIInstallRefreshesStaleSkillWithoutForce(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	if output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--local"); exitCode != 0 {
		t.Fatalf("first dflow agent --install --local exited %d, want 0\n%s", exitCode, output)
	}
	doc := expectedAgentDoc(t, repo)
	skill := projectSkillPath(repo, ".agents")
	requireFileContent(t, skill, doc)

	// Simulate a skill installed by an older dflow, with no document on disk, so
	// the document guard cannot be what makes the rerun succeed.
	writeRepoFile(t, skill, "---\nname: dflow\n---\n\n# an older dflow document\n")
	if err := os.Remove(filepath.Join(repo, agentWorkflowDefaultPath)); err != nil {
		t.Fatalf("failed to remove the document for the stale-skill pin: %v", err)
	}

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--local")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install --local over a stale skill exited %d, want 0\n%s", exitCode, output)
	}

	requireFileContent(t, skill, doc)
	requireSingleSkillFile(t, filepath.Dir(skill))
	requireFileAbsent(t, filepath.Join(home, ".agents"))
}

// --install --json reports the skill plan and still writes nothing: no document,
// no reference, no skill directory, and no key claiming a change --json cannot
// know about without writing.
func TestAgentCLIInstallJSONReportsPlanAndWritesNothing(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--json")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install --json exited %d, want 0\n%s", exitCode, output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	skills := requireJSONArray(t, doc, "skills")
	if len(skills) != 1 {
		t.Fatalf("skills has %d entries, want 1 (the portable root):\n%v", len(skills), skills)
	}
	entry, ok := skills[0].(map[string]any)
	if !ok {
		t.Fatalf("skills[0] = %#v, want an object", skills[0])
	}
	if len(entry) != 4 {
		t.Fatalf("skills[0] carries %d keys, want exactly root, dir, agents and exists: %v", len(entry), entry)
	}
	requireJSONString(t, entry, "root", "~/.agents/skills")
	requireJSONString(t, entry, "dir", filepath.Join("~/.agents/skills", agent.SkillName))
	requireJSONStringSlice(t, entry, "agents", []string{"pi"})
	requireJSONBool(t, entry, "exists", false)
	if _, ok := entry["changed"]; ok {
		t.Fatalf("skills[0] carries a changed field, which --json cannot know without writing: %v", entry)
	}

	requireFileAbsent(t, filepath.Join(repo, ".agents"))
	requireFileAbsent(t, filepath.Join(repo, "AGENTS.md"))
	requireFileAbsent(t, filepath.Join(home, ".agents"))
}

// The skills report is a fact about the filesystem, so an already installed skill
// is reported as existing -- and --json still writes nothing at all, not even a
// refresh of the file it reported.
func TestAgentCLIInstallJSONReportsAnInstalledSkill(t *testing.T) {
	home := skillInstallEnv(t)
	binary := buildDflowCLI(t)
	repo := newAgentRepo(t)

	installed := "---\nname: dflow\n---\n\n# installed by an earlier run\n"
	path := userSkillPath(home, ".agents")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create %s: %v", filepath.Dir(path), err)
	}
	writeRepoFile(t, path, installed)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, "agent", "--install", "--json")
	if exitCode != 0 {
		t.Fatalf("dflow agent --install --json exited %d, want 0\n%s", exitCode, output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	skills := requireJSONArray(t, doc, "skills")
	if len(skills) != 1 {
		t.Fatalf("skills has %d entries, want 1 (the portable root):\n%v", len(skills), skills)
	}
	entry, ok := skills[0].(map[string]any)
	if !ok {
		t.Fatalf("skills[0] = %#v, want an object", skills[0])
	}
	requireJSONBool(t, entry, "exists", true)

	requireFileContent(t, path, installed)
	requireFileAbsent(t, filepath.Join(repo, ".agents"))
}

// --local only chooses the scope of an install, so on its own it is a request
// with no effect: it must fail, name the flag that gives it one, and write
// nothing at all.
func TestAgentCLILocalWithoutInstallFailsAndWritesNothing(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantJSON bool
	}{
		{name: "human output", args: []string{"agent", "--local"}},
		{name: "json output", args: []string{"agent", "--local", "--json"}, wantJSON: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := skillInstallEnv(t)
			binary := buildDflowCLI(t)
			repo := newAgentRepo(t)

			output, exitCode := startCLIRawOutput(t, 30*time.Second, repo, binary, tc.args...)
			if exitCode == 0 {
				t.Fatalf("dflow %v exited 0, want non-zero\n%s", tc.args, output)
			}

			if tc.wantJSON {
				assertNoHumanChrome(t, output)
				doc := decodeSingleJSONDocument(t, output)
				message, ok := doc["error"].(string)
				if !ok || !strings.Contains(message, "--install") {
					t.Fatalf("the failure document must name --install, got: %v", doc)
				}
			} else if !strings.Contains(output, "--install") {
				t.Fatalf("the failure must name the flag that makes --local meaningful, got:\n%s", output)
			}

			for _, path := range []string{
				filepath.Join(repo, agentWorkflowDefaultPath),
				filepath.Join(repo, ".agents"),
				filepath.Join(repo, "AGENTS.md"),
				filepath.Join(repo, ".claude"),
				filepath.Join(home, ".agents"),
				filepath.Join(home, ".claude"),
			} {
				requireFileAbsent(t, path)
			}
		})
	}
}

// TestInitCLIWiresAgentReferences pins `dflow init` onto the shared writer in
// auto mode: the document is written, AGENTS.md carries exactly one block, and
// CLAUDE.md is not created.
func TestInitCLIWiresAgentReferences(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := initTempGitRepo(t)

	output, exitCode := runInteractiveCLI(t, 60*time.Second, repo, binary, initTerminalAnswers, "init")
	if exitCode != 0 {
		t.Fatalf("dflow init exited %d, want 0\n%s", exitCode, output)
	}

	requireFileContent(t, filepath.Join(repo, agentWorkflowDefaultPath), expectedAgentDoc(t, repo))
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
	requireFileAbsent(t, filepath.Join(repo, "CLAUDE.md"))
}

// TestInitCLIUpdatesExistingClaudeMd pins the auto-mode behaviour init gains for
// free: a pre-existing CLAUDE.md is refreshed, and AGENTS.md still carries
// exactly one block.
func TestInitCLIUpdatesExistingClaudeMd(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	repo := initTempGitRepo(t)

	existing := "# Claude project instructions\n"
	claudePath := filepath.Join(repo, "CLAUDE.md")
	writeRepoFile(t, claudePath, existing)

	output, exitCode := runInteractiveCLI(t, 60*time.Second, repo, binary, initTerminalAnswers, "init")
	if exitCode != 0 {
		t.Fatalf("dflow init exited %d, want 0\n%s", exitCode, output)
	}

	requireFileContent(t, claudePath, existing+"\n"+expectedReferenceBlock(agentWorkflowDefaultPath))
	requireFileContent(t, filepath.Join(repo, "AGENTS.md"), expectedReferenceBlock(agentWorkflowDefaultPath))
}

// terminalAnswer is one reactive answer to an interactive command: once trigger
// appears in the command's output, reply is typed into its terminal.
type terminalAnswer struct {
	trigger string
	reply   string
}

// initTerminalAnswers drives `dflow init` through its prompts, in order: three
// line answers accepted as their defaults (main, develop, uat), Enter on the
// default merge mode, Enter on an empty exception list, "n" to the push
// confirmation — the test repository has no origin and a test must never push —
// and Enter to accept the default "yes" on the agent-workflow confirmation.
//
// The triggers are the questions init renders. They are matched on their
// distinctive prefix rather than the whole sentence, but they are still a pin
// on init's user-visible prompts: rewording one makes the driver wait for an
// answer that never comes and the run fails with the captured transcript.
var initTerminalAnswers = []terminalAnswer{
	{trigger: "Main branch name:", reply: "\n"},
	{trigger: "Development branch name:", reply: "\n"},
	{trigger: "UAT branch name:", reply: "\n"},
	{trigger: "How do you manage merges by default in this project?", reply: "\n"},
	{trigger: "Which branches should behave differently", reply: "\n"},
	{trigger: "Do you want to push the base branches to 'origin'?", reply: "n\n"},
	{trigger: "Generate an agent workflow file", reply: "\n"},
}

// The cursor-position handshake survey's prompt renderer performs: it asks the
// terminal where the cursor is (DSR) to learn the terminal size, and blocks
// until the terminal answers. A pseudo-terminal allocated by script(1) has no
// terminal emulator behind it, so this driver plays that part with a fixed
// 80x24 report for the bottom-right corner.
const (
	dsrQuery  = "\x1b[6n"
	dsrReport = "\x1b[24;80R"
)

// runInteractiveCLI drives an interactive dflow command through a
// pseudo-terminal and returns its combined output and exit code.
//
// `dflow init` refuses to run unless both its stdin and stdout are terminals,
// so a pipe cannot exercise it and the package's ordinary helpers cannot reach
// its prompts. The pty comes from the host's script(1), the only way to
// allocate one on macOS and Linux without adding a dependency; a host without
// script(1) skips the pin instead of pretending it passed. The prompts are
// answered by terminalDriver, which also answers the cursor queries script(1)
// cannot.
//
// The run is bounded by timeout: a prompt the driver fails to satisfy fails the
// test with the captured transcript instead of hanging it.
func runInteractiveCLI(t *testing.T, timeout time.Duration, repo, binary string, answers []terminalAnswer, args ...string) (string, int) {
	t.Helper()

	scriptPath, err := exec.LookPath("script")
	if err != nil {
		t.Skipf("script(1) is required to drive an interactive command through a pseudo-terminal: %v", err)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// BSD script takes the command as trailing arguments.
		cmd = exec.Command(scriptPath, append([]string{"-q", "/dev/null", binary}, args...)...)
	} else {
		// util-linux script takes the command as one shell string. It is asked to
		// flush after every write: with its stdout being a pipe rather than the
		// terminal it records, an unbuffered script can hold a prompt in its own
		// buffer while the command it runs is already blocked on the answer to
		// that prompt. -e makes script report the command's own exit status.
		cmd = exec.Command(scriptPath, "-q", "-e", "-f", "-c", shellCommand(binary, args...), "/dev/null")
	}
	cmd.Dir = repo

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to open the interactive command's input: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to open the interactive command's output: %v", err)
	}
	// Prompts are written to stdout; folding stderr into the same stream leaves
	// one transcript to read and one to report on failure.
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start %s: %v", filepath.Base(binary), err)
	}

	driver := &terminalDriver{answers: answers}
	go driver.drive(stdin, stdout)

	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()

	select {
	case waitErr := <-waited:
		output := driver.transcript()
		if waitErr == nil {
			return output, 0
		}
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) {
			t.Fatalf("%s %v did not run: %v\n%s", filepath.Base(binary), args, waitErr, output)
		}
		return output, exitErr.ExitCode()
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		t.Fatalf("%s %v did not finish within %s; the driver never answered one of its prompts:\n%s", filepath.Base(binary), args, timeout, driver.transcript())
	}
	return "", 0
}

// terminalDriver answers an interactive child as its output arrives.
//
// It keeps the whole transcript for diagnostics, replies to every cursor query,
// and hands each configured answer over only once its trigger has been seen, in
// order. Answers are therefore delivered reactively; nothing depends on how
// fast the child renders or on the pty buffering input for it.
type terminalDriver struct {
	answers []terminalAnswer

	mu         sync.Mutex
	output     strings.Builder
	pending    string
	nextAnswer int
}

// drive reads the child's output until it ends or the process goes away.
func (d *terminalDriver) drive(input io.Writer, output io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := output.Read(buf)
		if n > 0 {
			d.consume(input, string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

// transcript returns everything the child has written so far.
func (d *terminalDriver) transcript() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.output.String()
}

// consume folds one chunk of child output into the transcript and answers
// whatever that chunk made answerable.
func (d *terminalDriver) consume(input io.Writer, chunk string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.output.WriteString(chunk)
	d.pending += chunk

	for strings.Contains(d.pending, dsrQuery) {
		d.pending = strings.Replace(d.pending, dsrQuery, "", 1)
		d.reply(input, dsrReport)
	}
	// Keep only a trailing fragment of a query that a chunk boundary may have
	// split, so the next chunk can complete it.
	d.pending = partialSuffix(d.pending, dsrQuery)

	for d.nextAnswer < len(d.answers) && strings.Contains(d.output.String(), d.answers[d.nextAnswer].trigger) {
		d.reply(input, d.answers[d.nextAnswer].reply)
		d.nextAnswer++
	}
}

// replyGap is the pause between two replies typed into the child's terminal.
//
// A real terminal answers each query the instant it arrives and is never asked
// two questions at once, which is the input survey's cursor reader is built for:
// it reads until it finds one position report and drops whatever else came with
// it in the same read. Sending our replies back to back makes that dropped tail
// real — two reports can land in one read, the second is discarded, and the
// command blocks forever on a prompt no terminal would ever fail to answer.
// Pacing the replies keeps this driver as slow as the child it is talking to,
// so every reply is consumed by the read the child is already blocked in before
// the next one is typed.
const replyGap = 50 * time.Millisecond

// reply types one reply into the child's terminal and pauses before the next.
// An error is ignored: a reply that arrives after the child closed its input is
// not a test failure.
func (d *terminalDriver) reply(input io.Writer, reply string) {
	_, _ = io.WriteString(input, reply)
	time.Sleep(replyGap)
}

// partialSuffix returns the longest suffix of s that is a proper prefix of
// sequence, which is what a later chunk may still complete.
func partialSuffix(s, sequence string) string {
	for length := len(sequence) - 1; length > 0; length-- {
		if len(s) >= length && strings.HasSuffix(s, sequence[:length]) {
			return s[len(s)-length:]
		}
	}
	return ""
}

// shellCommand joins program and args into one shell-safe command line for the
// util-linux script(1) form.
func shellCommand(program string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(program))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

// shellQuote single-quotes a shell argument, escaping embedded single quotes.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
