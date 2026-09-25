package agent

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// This file pins the two halves of skill installation: which directories a
// selection and a scope resolve to (PlanSkillPlacements), and the file-level
// rules the writer honours (InstallSkill).
//
// Every assertion is about a user-visible fact — a directory, a byte sequence, a
// file mode, an error's identity — rather than about the shape of the
// implementation, and the discriminating ones are chosen so a plausible wrong
// implementation fails them: a caller-ordered plan instead of a registry-ordered
// one, a rewrite of identical bytes instead of a no-op, an unresolved "~", a
// leftover temporary file.

// skillFileContent is the content the install tests write: the opening of the
// generated document, recognisable in a failure message.
const skillFileContent = "---\nname: dflow\n---\n\n# dflow Workflow\n"

// installedSkillDir is a fresh, nested skills directory no run has touched yet.
func installedSkillDir(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), ".agents", "skills", SkillName)
}

// redirectHome points HOME at a fresh directory for the duration of the test.
//
// Every assertion about a "~"-prefixed root must go through it: the suite must
// never resolve — let alone write — into the developer's real ~/.agents/skills
// or ~/.claude/skills.
func redirectHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// requireInstalledSkill asserts path holds exactly content as a regular file
// with the given permission bits.
func requireInstalledSkill(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("the installed skill file %s does not exist: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("%s mode = %v, want a regular file", path, info.Mode())
	}
	if got := info.Mode().Perm(); got != mode {
		t.Errorf("%s mode = %04o, want %04o", path, got, mode)
	}
	if got := readFile(t, path); got != content {
		t.Errorf("%s =\n%q\nwant\n%q", path, got, content)
	}
}

// requireNoPath asserts nothing exists at path, whatever the error flavour.
func requireNoPath(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("%s exists, want it untouched", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

// placementFor is the placement one root must resolve to, with its agents.
func placementFor(root string, agents ...AgentID) SkillPlacement {
	return SkillPlacement{
		Root:   root,
		Dir:    filepath.Join(root, SkillName),
		Agents: agents,
	}
}

// requirePlacements asserts the whole plan, order included: a plan that names
// the right directories in the wrong order is a different plan.
func requirePlacements(t *testing.T, got, want []SkillPlacement) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PlanSkillPlacements() =\n%v\nwant\n%v", got, want)
	}
}

// The three agents that share .agents/skills produce ONE placement, not three:
// one file serving them all is the entire point of the portable directory.
func TestPlanSkillPlacementsDedupesTheSharedRoot(t *testing.T) {
	requirePlacements(t, PlanSkillPlacements(Agents(), true), []SkillPlacement{
		placementFor(".agents/skills", AgentPi, AgentCodex, AgentOpenCode),
		placementFor(".claude/skills", AgentClaude),
	})
}

// The scope chooses the root: the user scope is where a skill is discoverable
// from every project, the local one keeps it inside the repository.
func TestPlanSkillPlacementsScopeSelectsTheRoot(t *testing.T) {
	agents := Agents()

	requirePlacements(t, PlanSkillPlacements(agents, false), []SkillPlacement{
		placementFor("~/.agents/skills", AgentPi),
		placementFor("~/.codex/skills", AgentCodex),
		placementFor("~/.config/opencode/skills", AgentOpenCode),
		placementFor("~/.claude/skills", AgentClaude),
	})

	// The same selection in the other scope must not reuse a single one of those
	// directories: the user roots are private copies, the local roots are the
	// project's.
	local := PlanSkillPlacements(agents, true)
	user := PlanSkillPlacements(agents, false)
	for _, placement := range local {
		for _, other := range user {
			if placement.Root == other.Root {
				t.Fatalf("scope did not change the root: both plans target %q", placement.Root)
			}
		}
	}
}

// Naming claude places claude's root only, in whichever scope is asked for.
func TestPlanSkillPlacementsClaudeAlone(t *testing.T) {
	claude := []Agent{mustLookup(t, AgentClaude)}

	requirePlacements(t, PlanSkillPlacements(claude, false), []SkillPlacement{
		placementFor("~/.claude/skills", AgentClaude),
	})
	requirePlacements(t, PlanSkillPlacements(claude, true), []SkillPlacement{
		placementFor(".claude/skills", AgentClaude),
	})
}

// Auto mode places the PORTABLE root and nothing else. The portable agent is
// pi's registry entry, so this pins the fact the CLI's auto selection rests on:
// pi's roots are the shared .agents/skills ones, which is why a plain
// `dflow agent --install` writes one file into $HOME instead of four.
func TestPlanSkillPlacementsPortableSelection(t *testing.T) {
	portable := []Agent{mustLookup(t, AgentPi)}

	requirePlacements(t, PlanSkillPlacements(portable, false), []SkillPlacement{
		placementFor("~/.agents/skills", AgentPi),
	})
	requirePlacements(t, PlanSkillPlacements(portable, true), []SkillPlacement{
		placementFor(".agents/skills", AgentPi),
	})
}

// Placements follow the REGISTRY, not the caller's order: the same selection
// reversed must produce the same plan, because the plan is a fact about dflow's
// agent table rather than about how a caller happened to build its slice.
func TestPlanSkillPlacementsFollowsRegistryOrder(t *testing.T) {
	agents := Agents()
	reversed := make([]Agent, 0, len(agents))
	for i := len(agents) - 1; i >= 0; i-- {
		reversed = append(reversed, agents[i])
	}

	requirePlacements(t, PlanSkillPlacements(reversed, false), []SkillPlacement{
		placementFor("~/.agents/skills", AgentPi),
		placementFor("~/.codex/skills", AgentCodex),
		placementFor("~/.config/opencode/skills", AgentOpenCode),
		placementFor("~/.claude/skills", AgentClaude),
	})
}

// An agent with no directory for the requested scope contributes nothing, and
// the scope decides which directory that is.
func TestPlanSkillPlacementsSkipsAgentsWithoutADirectoryForTheScope(t *testing.T) {
	t.Run("agent with no directory at all", func(t *testing.T) {
		agents := []Agent{
			{ID: AgentID("ghost")},
			mustLookup(t, AgentPi),
		}

		requirePlacements(t, PlanSkillPlacements(agents, true), []SkillPlacement{
			placementFor(".agents/skills", AgentPi),
		})
		requirePlacements(t, PlanSkillPlacements(agents, false), []SkillPlacement{
			placementFor("~/.agents/skills", AgentPi),
		})
	})

	t.Run("registry agent with a blanked user root", func(t *testing.T) {
		agents := Agents()
		for i := range agents {
			if agents[i].ID == AgentCodex {
				agents[i].UserSkillDir = ""
			}
		}

		requirePlacements(t, PlanSkillPlacements(agents, false), []SkillPlacement{
			placementFor("~/.agents/skills", AgentPi),
			placementFor("~/.config/opencode/skills", AgentOpenCode),
			placementFor("~/.claude/skills", AgentClaude),
		})
		// The blanking is per scope: codex's project root is untouched, so the
		// local plan still groups all three shared-root agents into one placement.
		requirePlacements(t, PlanSkillPlacements(agents, true), []SkillPlacement{
			placementFor(".agents/skills", AgentPi, AgentCodex, AgentOpenCode),
			placementFor(".claude/skills", AgentClaude),
		})
	})
}

// Rule 1: an absent file is created, parents and all, with mode 0644.
func TestInstallSkillCreatesAbsentFile(t *testing.T) {
	dir := installedSkillDir(t)

	changed, path, err := InstallSkill(dir, []byte(skillFileContent))
	if err != nil {
		t.Fatalf("InstallSkill() error = %v, want nil", err)
	}
	if !changed {
		t.Errorf("InstallSkill() changed = false, want true for a file it created")
	}
	if want := filepath.Join(dir, SkillFileName); path != want {
		t.Errorf("InstallSkill() path = %q, want %q", path, want)
	}
	requireInstalledSkill(t, path, skillFileContent, 0644)

	// The installation is self-contained: the skill directory and its parents are
	// created by the install, not by the caller.
	for _, parent := range []string{
		filepath.Dir(dir),
		filepath.Dir(filepath.Dir(dir)),
	} {
		info, err := os.Stat(parent)
		if err != nil {
			t.Fatalf("InstallSkill() did not create %s: %v", parent, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory after InstallSkill()", parent)
		}
	}
}

// Rule 2: identical bytes are a no-op. The pre-existing mode is what makes
// "untouched" observable: a rewrite through a temporary file would restore 0644.
func TestInstallSkillLeavesIdenticalBytesUntouched(t *testing.T) {
	dir := installedSkillDir(t)
	path := filepath.Join(dir, SkillFileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(skillFileContent), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}

	changed, gotPath, err := InstallSkill(dir, []byte(skillFileContent))
	if err != nil {
		t.Fatalf("InstallSkill() error = %v, want nil", err)
	}
	if changed {
		t.Errorf("InstallSkill() changed = true, want false for identical bytes")
	}
	if gotPath != path {
		t.Errorf("InstallSkill() path = %q, want %q", gotPath, path)
	}

	requireInstalledSkill(t, path, skillFileContent, 0600)
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("%s modification time moved from %s to %s, so an identical install rewrote the file", path, before.ModTime(), after.ModTime())
	}
}

// Rule 3: different bytes are rewritten, with no --force in sight. The skill
// file is dflow's own artifact, so a stale copy is a bug rather than a hand edit
// to protect — which is why InstallSkill has no force concept at all.
func TestInstallSkillRewritesDifferentBytesWithoutForce(t *testing.T) {
	dir := installedSkillDir(t)
	path := filepath.Join(dir, SkillFileName)
	writeFile(t, path, "# an older dflow document\n")

	changed, gotPath, err := InstallSkill(dir, []byte(skillFileContent))
	if err != nil {
		t.Fatalf("InstallSkill() error = %v, want nil", err)
	}
	if !changed {
		t.Errorf("InstallSkill() changed = false, want true when the bytes differ")
	}
	if gotPath != path {
		t.Errorf("InstallSkill() path = %q, want %q", gotPath, path)
	}
	requireInstalledSkill(t, path, skillFileContent, 0644)
}

// Rule 4: a "~"-prefixed directory is resolved through the user's home
// directory, never left as a literal directory name.
func TestInstallSkillExpandsTildeThroughTheHomeDirectory(t *testing.T) {
	home := redirectHome(t)
	dir := "~/.agents/skills/" + SkillName

	changed, path, err := InstallSkill(dir, []byte(skillFileContent))
	if err != nil {
		t.Fatalf("InstallSkill(%q) error = %v, want nil", dir, err)
	}
	if !changed {
		t.Errorf("InstallSkill() changed = false, want true for a file it created")
	}
	want := filepath.Join(home, ".agents", "skills", SkillName, SkillFileName)
	if path != want {
		t.Errorf("InstallSkill(%q) path = %q, want %q", dir, path, want)
	}
	requireInstalledSkill(t, want, skillFileContent, 0644)

	// An unresolved tilde would have created a directory literally named "~"
	// under the test's working directory.
	requireNoPath(t, filepath.Join("~", ".agents", "skills", SkillName, SkillFileName))
}

// SkillFilePath is the one resolver the writer and a read-only reporter share,
// so it must accept both registry forms and name the file inside the directory.
func TestSkillFilePathResolvesBothRegistryForms(t *testing.T) {
	home := redirectHome(t)

	userPath, err := SkillFilePath("~/.claude/skills/" + SkillName)
	if err != nil {
		t.Fatalf("SkillFilePath() error = %v, want nil", err)
	}
	if want := filepath.Join(home, ".claude", "skills", SkillName, SkillFileName); userPath != want {
		t.Errorf("SkillFilePath() = %q, want %q", userPath, want)
	}

	projectPath, err := SkillFilePath(".agents/skills/" + SkillName)
	if err != nil {
		t.Fatalf("SkillFilePath() error = %v, want nil", err)
	}
	abs, err := filepath.Abs(filepath.Join(".agents", "skills", SkillName, SkillFileName))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if projectPath != abs {
		t.Errorf("SkillFilePath() = %q, want the absolute form %q", projectPath, abs)
	}
}

// Rule 5: the write is atomic, so what it must never leave behind is a
// temporary file — on the create path or on the rewrite path.
func TestInstallSkillLeavesNoTemporaryFileBehind(t *testing.T) {
	dir := installedSkillDir(t)

	if _, _, err := InstallSkill(dir, []byte(skillFileContent)); err != nil {
		t.Fatalf("InstallSkill() error = %v, want nil", err)
	}
	if _, _, err := InstallSkill(dir, []byte(skillFileContent+"\nsecond version\n")); err != nil {
		t.Fatalf("InstallSkill() rewrite error = %v, want nil", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != 1 || names[0] != SkillFileName {
		t.Fatalf("%s holds %v, want exactly [%s]", dir, names, SkillFileName)
	}
}

// Rule 5, the mechanism half: the replacement goes through a rename, which is
// what makes it atomic. A rename hands the target the temporary file's identity,
// so the inode changes; truncating and rewriting the file in place would keep it
// — and would be exactly the non-atomic write the rule exists to forbid.
//
// The check is POSIX-only because inode numbers are.
func TestInstallSkillReplacesTheFileRatherThanTruncatingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("inode numbers are a POSIX filesystem detail")
	}

	dir := installedSkillDir(t)
	path := filepath.Join(dir, SkillFileName)
	writeFile(t, path, "# an older dflow document\n")

	before := inodeOf(t, path)
	if _, _, err := InstallSkill(dir, []byte(skillFileContent)); err != nil {
		t.Fatalf("InstallSkill() error = %v, want nil", err)
	}
	after := inodeOf(t, path)

	if before == after {
		t.Errorf("the installed file kept inode %d, so it was rewritten in place instead of being replaced atomically", before)
	}
}

// inodeOf returns the inode number of path.
func inodeOf(t *testing.T, path string) uint64 {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat %s returned %T, want *syscall.Stat_t", path, info.Sys())
	}
	return stat.Ino
}

// Rule 6: failures are wrapped, naming the operation and the path and keeping
// the underlying error reachable through errors.Is.
func TestInstallSkillWrapsFilesystemErrors(t *testing.T) {
	content := []byte(skillFileContent)

	t.Run("a regular file where the directory must be", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocked")
		writeFile(t, blocker, "not a directory\n")
		dir := filepath.Join(blocker, "skills", SkillName)

		changed, path, err := InstallSkill(dir, content)
		if err == nil {
			t.Fatalf("InstallSkill(%q) error = nil, want a failure", dir)
		}
		if changed {
			t.Errorf("InstallSkill() changed = true alongside an error, want false")
		}
		if want := filepath.Join(dir, SkillFileName); path != want {
			t.Errorf("InstallSkill() path = %q, want the failing target %q", path, want)
		}
		if !strings.Contains(err.Error(), dir) {
			t.Errorf("error %q must name the directory it failed on (%s)", err, dir)
		}
		if !errors.Is(err, syscall.ENOTDIR) {
			t.Errorf("error %q must wrap the underlying ENOTDIR", err)
		}
	})

	t.Run("the target is a directory", func(t *testing.T) {
		dir := installedSkillDir(t)
		writeFile(t, filepath.Join(dir, SkillFileName, "keep"), "the target path is a directory\n")

		changed, _, err := InstallSkill(dir, content)
		if err == nil {
			t.Fatalf("InstallSkill(%q) error = nil, want a failure", dir)
		}
		if changed {
			t.Errorf("InstallSkill() changed = true alongside an error, want false")
		}
		if !errors.Is(err, syscall.EISDIR) {
			t.Errorf("error %q must wrap the underlying EISDIR", err)
		}
	})
}
