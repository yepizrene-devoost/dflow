// This file is the pure, fact-encoding half of the discovery layer: it maps the
// agents dflow supports to the instruction files they read and the skill
// directories they discover. It performs no file I/O so the table can be
// reasoned about and tested without a filesystem.
package agent

import (
	"fmt"
	"sort"
	"strings"
)

// AgentID is the stable identifier for a supported agent.
type AgentID string

// The supported agents.
const (
	AgentPi       AgentID = "pi"
	AgentCodex    AgentID = "codex"
	AgentOpenCode AgentID = "opencode"
	AgentClaude   AgentID = "claude"
)

// CanonicalInstructionFile is the agent-agnostic instruction file dflow always
// ensures, creating it when the project has none. It is the single file pi,
// codex and opencode all read, so wiring it wires three agents at once.
const CanonicalInstructionFile = "AGENTS.md"

// AgentSpecAll is the --agents value that selects every supported agent.
const AgentSpecAll = "all"

// Agent is one supported coding agent and the discovery surfaces dflow targets
// for it.
type Agent struct {
	ID          AgentID
	DisplayName string
	// Instructions lists the instruction files this agent reads, in discovery
	// order (earlier files win when the agent merges several).
	Instructions []string
	// SkillDir is the ONE project-relative skills directory dflow prefers to
	// install into for this agent: the spec-portable Agent Skills directory when
	// the agent reads several. It is not a claim that the agent reads only that
	// directory.
	SkillDir string
	// UserSkillDir is the user-scope skills directory, "~"-prefixed.
	UserSkillDir string
}

// registry is the verified discovery table, in stable CLI order. It is
// package-private and must never be returned directly: callers receive deep
// copies so a mutation cannot corrupt the next call.
var registry = []Agent{
	{
		ID:           AgentPi,
		DisplayName:  "Pi",
		Instructions: []string{"AGENTS.md", "CLAUDE.md"},
		SkillDir:     ".agents/skills",
		UserSkillDir: "~/.agents/skills",
	},
	{
		ID:           AgentCodex,
		DisplayName:  "Codex",
		Instructions: []string{"AGENTS.md"},
		SkillDir:     ".agents/skills",
		UserSkillDir: "~/.codex/skills",
	},
	{
		ID:           AgentOpenCode,
		DisplayName:  "opencode",
		Instructions: []string{"AGENTS.md"},
		SkillDir:     ".agents/skills",
		UserSkillDir: "~/.config/opencode/skills",
	},
	{
		ID:           AgentClaude,
		DisplayName:  "Claude Code",
		Instructions: []string{"CLAUDE.md"},
		SkillDir:     ".claude/skills",
		UserSkillDir: "~/.claude/skills",
	},
}

// Agents returns the registry in stable CLI order: pi, codex, opencode, claude.
// The result is a deep copy; mutating it never corrupts the registry.
func Agents() []Agent {
	return copyAgents(registry)
}

// LookupAgent resolves an agent id against the registry. The returned Agent is a
// copy, so its instruction slice does not alias the registry either.
func LookupAgent(id AgentID) (Agent, bool) {
	for _, a := range registry {
		if a.ID == id {
			return copyAgent(a), true
		}
	}
	return Agent{}, false
}

// ParseAgentSpec resolves an --agents value.
//
//	""          -> every agent       (auto; see below)
//	"all"       -> every agent
//	"pi,claude" -> those agents, deduped
//
// The empty string selects every supported agent: auto is not a special value
// of this function, it is expressed by the caller's explicit flag when it calls
// PlanInstructionTargets. Because "" returns Agents() rather than nil, the two
// functions compose totally:
//
//	agents, _ := ParseAgentSpec(spec)
//	plan := PlanInstructionTargets(agents, exists, explicit)
//
// with explicit == false for an auto run (spec == "") and explicit == true when
// the user named agents. The caller distinguishes the two; ParseAgentSpec does
// not, so a forgotten error cannot silently degrade auto mode to an empty
// selection.
//
// Unknown ids and specs that resolve to zero agents are errors naming the
// culprit. Elements are whitespace-trimmed and empty elements are ignored; a
// whitespace-only spec is therefore a zero-resolution error, not auto. The
// result is ordered by the registry, not by the order written, and never aliases
// it.
func ParseAgentSpec(spec string) ([]Agent, error) {
	if spec == "" {
		return Agents(), nil
	}
	trimmed := strings.TrimSpace(spec)
	if trimmed == AgentSpecAll {
		return Agents(), nil
	}

	requested := make(map[AgentID]bool)
	for _, raw := range strings.Split(trimmed, ",") {
		id := AgentID(strings.TrimSpace(raw))
		if id == "" {
			continue
		}
		if _, ok := LookupAgent(id); !ok {
			return nil, fmt.Errorf("unknown agent %q (valid agents: %s, %s)", id, agentIDList(), AgentSpecAll)
		}
		requested[id] = true
	}

	selected := make([]Agent, 0, len(requested))
	for _, a := range registry {
		if requested[a.ID] {
			selected = append(selected, copyAgent(a))
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("agent spec %q resolved to no agents", spec)
	}
	return selected, nil
}

// agentIDList renders the known ids for error messages, in registry order.
func agentIDList() string {
	ids := make([]string, 0, len(registry))
	for _, a := range registry {
		ids = append(ids, string(a.ID))
	}
	return strings.Join(ids, ", ")
}

// InstructionTarget is one instruction file that must carry the workflow
// reference.
type InstructionTarget struct {
	// File is project-relative and slash-separated.
	File string
	// Agents is a registry fact, not a projection of any selection: it lists
	// every registry agent that reads File, in registry order, whatever set of
	// agents a caller selected. It is therefore never empty. AGENTS.md is always
	// [pi, codex, opencode]; CLAUDE.md is always [pi, claude]. A caller must not
	// read it as the set of agents this run named.
	Agents []AgentID
	// Create is true when the file does not exist and must be created.
	Create bool
}

// PlanInstructionTargets resolves agents to the instruction files that must
// carry the reference. The caller names the selection through agents and says
// whether it was explicit (--agents named them) or auto (spec == "", which
// ParseAgentSpec resolves to every agent).
//
// Inclusion rules:
//
//   - CanonicalInstructionFile is in the plan always, EXCEPT when explicit is
//     true and none of the selected agents reads it. So --agents claude does not
//     create AGENTS.md.
//   - Any other registry instruction file is in the plan when exists(file) is
//     true, OR when explicit is true and file is the PRIMARY instruction file of
//     a selected agent. PRIMARY is the first entry of that agent's Instructions:
//     pi's is AGENTS.md, claude's is CLAUDE.md. So --agents claude creates
//     CLAUDE.md, but --agents pi does NOT create CLAUDE.md even though pi reads
//     it: the file a named agent's instructions canonically live in is what
//     "named" authorizes creating, and a file the agent merely also reads is
//     maintained only when it already exists. No dflow run drops a surprise
//     CLAUDE.md into a repository that does not use Claude Code.
//
// Create is !exists(file). The order is deterministic: the canonical file first,
// then the rest sorted by name. A nil exists func is a programming error.
func PlanInstructionTargets(agents []Agent, exists func(file string) bool, explicit bool) []InstructionTarget {
	if exists == nil {
		panic("agent: PlanInstructionTargets called with a nil exists func")
	}

	// Normalise the selection to a set keyed by id, dropping duplicates.
	selected := make(map[AgentID]bool, len(agents))
	for _, a := range agents {
		selected[a.ID] = true
	}

	// readers is the registry fact: every registry agent that reads a file,
	// independent of what the caller selected. Iterating the registry keeps the
	// lists in registry order.
	readers := map[string][]AgentID{}
	for _, a := range registry {
		for _, file := range a.Instructions {
			readers[file] = appendUniqueAgent(readers[file], a.ID)
		}
	}

	// selectedReads reports whether any selected agent reads file.
	selectedReads := func(file string) bool {
		for _, id := range readers[file] {
			if selected[id] {
				return true
			}
		}
		return false
	}

	// primaryFiles is the set of files that are the PRIMARY instruction file of
	// a selected agent: the first entry of its Instructions.
	primaryFiles := map[string]bool{}
	for _, a := range agents {
		if len(a.Instructions) > 0 {
			primaryFiles[a.Instructions[0]] = true
		}
	}

	order := make([]string, 0, len(readers))
	if !explicit || selectedReads(CanonicalInstructionFile) {
		order = append(order, CanonicalInstructionFile)
	}

	var otherFiles []string
	for file := range readers {
		if file == CanonicalInstructionFile {
			continue
		}
		if exists(file) || (explicit && primaryFiles[file]) {
			otherFiles = append(otherFiles, file)
		}
	}
	sort.Strings(otherFiles)
	order = append(order, otherFiles...)

	plan := make([]InstructionTarget, 0, len(order))
	for _, file := range order {
		plan = append(plan, InstructionTarget{
			File:   file,
			Agents: readers[file],
			Create: !exists(file),
		})
	}
	return plan
}

// appendUniqueAgent appends id when it is not already the last element. Agent
// ids are appended in registry order, so a simple adjacency check is enough.
func appendUniqueAgent(ids []AgentID, id AgentID) []AgentID {
	if len(ids) > 0 && ids[len(ids)-1] == id {
		return ids
	}
	return append(ids, id)
}

// copyAgents deep-copies a slice of agents, including each instruction slice.
func copyAgents(agents []Agent) []Agent {
	out := make([]Agent, len(agents))
	for i, a := range agents {
		out[i] = copyAgent(a)
	}
	return out
}

// copyAgent deep-copies one agent so its instruction slice does not alias the
// registry.
func copyAgent(a Agent) Agent {
	a.Instructions = append([]string(nil), a.Instructions...)
	return a
}
