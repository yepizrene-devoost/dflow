package agent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// This file is the placement half of the discovery layer: it turns a selection
// of agents and a scope into the distinct skill directories an install writes,
// and it writes the skill file into them.
//
// One document, several discovery adapters: the skill file's bytes are the
// generated workflow document and never a second rendering, so the skill an
// agent discovers and the reference an instruction file points at can never
// disagree.

// The skill artifact every supported agent discovers: a directory named after
// the skill, holding a single SKILL.md.
const (
	// SkillName is the skill directory's name, and the name the document's
	// frontmatter declares. It is the directory every agent looks for.
	SkillName = "dflow"
	// SkillFileName is the file inside that directory. Agents discover skills by
	// reading it, so its name is part of the artifact rather than a local choice.
	SkillFileName = "SKILL.md"
)

// SkillPlacement is one skill directory an install targets, together with every
// selected agent that discovers a skill there.
type SkillPlacement struct {
	// Root is the skills root exactly as the registry records it: "~"-prefixed
	// for the user scope, project-relative for the local one.
	Root string
	// Dir is Root joined with SkillName: the directory the skill file goes into.
	Dir string
	// Agents lists every selected agent that reads Root, in registry order. It is
	// never empty: a placement exists because at least one agent contributed it.
	Agents []AgentID
}

// PlanSkillPlacements resolves a selection of agents to the distinct skill
// directories an install must write to.
//
// local chooses the scope: each agent's project-relative directory (Agent.SkillDir)
// when true, its "~"-prefixed user directory (Agent.UserSkillDir) when false.
// Placements are deduplicated by directory, so the three agents that share
// .agents/skills produce ONE placement rather than three — one file serving them
// all is the point of the portable directory. They are ordered by the agents'
// position in the registry rather than by the caller's order, and an agent with
// no directory for the requested scope contributes nothing.
//
// The agent values are read from the caller's slice, never from the package
// registry table: the table is a fact about the agents, while this function
// reports what one specific selection would write, and a selection always
// belongs to the caller.
func PlanSkillPlacements(agents []Agent, local bool) []SkillPlacement {
	ordered := registryOrderedCopy(agents)

	index := make(map[string]int, len(ordered))
	placements := make([]SkillPlacement, 0, len(ordered))

	for _, a := range ordered {
		root := a.UserSkillDir
		if local {
			root = a.SkillDir
		}
		// A scope an agent has no directory for skips it: the caller asked for
		// that scope, and inventing a directory would write where the agent does
		// not look.
		if root == "" {
			continue
		}

		dir := filepath.Join(root, SkillName)
		if at, ok := index[dir]; ok {
			placements[at].Agents = appendUniqueAgent(placements[at].Agents, a.ID)
			continue
		}

		index[dir] = len(placements)
		placements = append(placements, SkillPlacement{
			Root:   root,
			Dir:    dir,
			Agents: []AgentID{a.ID},
		})
	}

	return placements
}

// registryOrderedCopy returns agents in registry order, which is the order
// placements and their agent lists are reported in.
//
// The caller's slice is copied, never reordered in place. An id the registry does
// not know keeps the caller's relative order after every known id: the ordering
// rule is about the agents dflow supports, and it must not drop an agent a caller
// constructed.
func registryOrderedCopy(agents []Agent) []Agent {
	rank := func(id AgentID) int {
		for i := range registry {
			if registry[i].ID == id {
				return i
			}
		}
		return len(registry)
	}

	ordered := make([]Agent, len(agents))
	copy(ordered, agents)
	sort.SliceStable(ordered, func(i, j int) bool { return rank(ordered[i].ID) < rank(ordered[j].ID) })
	return ordered
}

// SkillFilePath resolves a skill directory to the absolute path of the skill
// file inside it, expanding a leading "~" through the user's home directory.
//
// It is the ONE place a "~"-prefixed registry root becomes a real path, so the
// writer and a read-only reporter resolve the same file and cannot drift apart.
func SkillFilePath(dir string) (string, error) {
	resolved, err := expandHome(dir)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve the skill directory %s: %w", dir, err)
	}
	return filepath.Join(abs, SkillFileName), nil
}

// expandHome turns a "~"-prefixed directory into a real one, resolving the home
// directory per call so a test can point HOME somewhere disposable. Any other
// directory is already real and is returned unchanged.
func expandHome(dir string) (string, error) {
	if dir != "~" && !strings.HasPrefix(dir, "~/") {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve the home directory for %s: %w", dir, err)
	}
	if dir == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(dir, "~/")), nil
}

// InstallSkill writes content as the skill file inside dir, creating dir and its
// parents when they are missing, and reports whether the file changed.
//
// A file whose bytes are already content is left completely untouched and reports
// changed == false, which is what makes a second run a no-op. Different bytes are
// rewritten without any --force: unlike the workflow document, the skill file is
// dflow's own artifact and a stale copy is a bug rather than a local edit to
// protect.
//
// A rewrite preserves the mode of the file it replaces: because writeFileAtomic
// always applies the mode it is handed, a SKILL.md a user installed by hand with
// mode 0600 must not be widened to 0644 on every refresh. A file that does not
// exist yet has no prior mode to honor and is created at 0644.
//
// The write is atomic: content goes to a temporary file beside the target and
// moves into place with one rename, so an interrupted run can never leave a
// truncated SKILL.md behind — a reader sees either the previous file or the whole
// new one.
func InstallSkill(dir string, content []byte) (bool, string, error) {
	path, err := SkillFilePath(dir)
	if err != nil {
		return false, "", err
	}

	existing, readErr := os.ReadFile(path)
	if readErr == nil && bytes.Equal(existing, content) {
		return false, path, nil
	}
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, path, fmt.Errorf("read the installed skill file %s: %w", path, readErr)
	}

	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return false, path, fmt.Errorf("stat the installed skill file %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, path, fmt.Errorf("create the skill directory %s: %w", filepath.Dir(path), err)
	}
	if err := writeFileAtomic(path, content, mode); err != nil {
		return false, path, err
	}
	return true, path, nil
}

// writeFileAtomic writes content to path through a temporary file in the target's
// own directory, then renames it into place.
//
// The temporary file shares the target's directory so the rename stays within one
// filesystem and is therefore atomic; a temporary file anywhere else would turn
// the move into a copy. The mode is applied before the rename, so the target never
// exists with a wider mode than intended, and a temporary file never survives a
// failure. It lives here, next to the installer that needs it, so the instruction
// reference writer can adopt the same primitive instead of growing a second one.
func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("create a temporary file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	// Cleanup for every path that does not reach a successful rename; after one it
	// is a no-op, because the temporary name no longer exists.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set the mode of %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}

	return nil
}
