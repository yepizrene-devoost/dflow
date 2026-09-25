package agent

import (
	"reflect"
	"strings"
	"testing"
)

// wantRegistry is the verified discovery table this package must encode. It is
// spelled out here rather than derived from Agents() so the test pins facts
// instead of restating the implementation.
func wantRegistry() []Agent {
	return []Agent{
		{
			ID:           AgentPi,
			Instructions: []string{"AGENTS.md", "CLAUDE.md"},
			SkillDir:     ".agents/skills",
			UserSkillDir: "~/.agents/skills",
		},
		{
			ID:           AgentCodex,
			Instructions: []string{"AGENTS.md"},
			SkillDir:     ".agents/skills",
			UserSkillDir: "~/.codex/skills",
		},
		{
			ID:           AgentOpenCode,
			Instructions: []string{"AGENTS.md"},
			SkillDir:     ".agents/skills",
			UserSkillDir: "~/.config/opencode/skills",
		},
		{
			ID:           AgentClaude,
			Instructions: []string{"CLAUDE.md"},
			SkillDir:     ".claude/skills",
			UserSkillDir: "~/.claude/skills",
		},
	}
}

func TestAgentsRegistryOrderAndFields(t *testing.T) {
	got := Agents()
	want := wantRegistry()

	if len(got) != len(want) {
		t.Fatalf("Agents() returned %d agents, want %d", len(got), len(want))
	}
	wantOrder := []AgentID{AgentPi, AgentCodex, AgentOpenCode, AgentClaude}
	for i := range want {
		if got[i].ID != wantOrder[i] {
			t.Errorf("Agents()[%d].ID = %q, want %q", i, got[i].ID, wantOrder[i])
		}
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("Agents()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestAgentsSharedSurfaceIsEncoded(t *testing.T) {
	// The strategic fact the registry exists to record: one write into
	// .agents/skills and AGENTS.md serves pi, codex and opencode.
	for _, a := range Agents() {
		switch a.ID {
		case AgentPi, AgentCodex, AgentOpenCode:
			if a.SkillDir != ".agents/skills" {
				t.Errorf("agent %s SkillDir = %q, want .agents/skills", a.ID, a.SkillDir)
			}
			if !containsString(a.Instructions, "AGENTS.md") {
				t.Errorf("agent %s does not read AGENTS.md", a.ID)
			}
		case AgentClaude:
			if a.SkillDir != ".claude/skills" {
				t.Errorf("claude SkillDir = %q, want .claude/skills", a.SkillDir)
			}
			if containsString(a.Instructions, "AGENTS.md") {
				t.Error("claude is encoded as reading AGENTS.md, which it does not")
			}
		}
	}
}

func TestLookupAgent(t *testing.T) {
	for _, want := range wantRegistry() {
		got, ok := LookupAgent(want.ID)
		if !ok {
			t.Errorf("LookupAgent(%q) not found", want.ID)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("LookupAgent(%q) = %#v, want %#v", want.ID, got, want)
		}
	}

	if _, ok := LookupAgent("gemini"); ok {
		t.Error("LookupAgent(\"gemini\") reported a hit for an unknown agent")
	}
}

func TestAgentsReturnedSliceDoesNotAliasRegistry(t *testing.T) {
	first := Agents()
	first[0].Instructions[0] = "MUTATED.md"

	second := Agents()
	if second[0].Instructions[0] != "AGENTS.md" {
		t.Errorf("mutating the returned slice corrupted the registry: Instructions[0] = %q", second[0].Instructions[0])
	}
}

func TestParseAgentSpec(t *testing.T) {
	ids := func(agents []Agent) []AgentID {
		if agents == nil {
			return nil
		}
		out := make([]AgentID, len(agents))
		for i, a := range agents {
			out[i] = a.ID
		}
		return out
	}

	cases := []struct {
		name    string
		spec    string
		want    []AgentID
		wantErr string
	}{
		{name: "auto selects every agent", spec: "", want: []AgentID{AgentPi, AgentCodex, AgentOpenCode, AgentClaude}},
		{name: "all", spec: "all", want: []AgentID{AgentPi, AgentCodex, AgentOpenCode, AgentClaude}},
		{name: "all with whitespace", spec: "  all  ", want: []AgentID{AgentPi, AgentCodex, AgentOpenCode, AgentClaude}},
		{name: "list", spec: "pi,claude", want: []AgentID{AgentPi, AgentClaude}},
		{name: "registry order not written order", spec: "claude,pi", want: []AgentID{AgentPi, AgentClaude}},
		{name: "dedupe", spec: "pi,pi,pi", want: []AgentID{AgentPi}},
		{name: "whitespace and empties", spec: " pi , , claude ,", want: []AgentID{AgentPi, AgentClaude}},
		{name: "single", spec: "opencode", want: []AgentID{AgentOpenCode}},
		{name: "unknown", spec: "gemini", wantErr: "gemini"},
		{name: "unknown among known", spec: "pi,gemini", wantErr: "gemini"},
		{name: "zero resolution from empties", spec: ",", wantErr: "no agents"},
		{name: "zero resolution from spaces", spec: "   ", wantErr: "no agents"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAgentSpec(tc.spec)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseAgentSpec(%q) returned no error, want one naming %q", tc.spec, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("ParseAgentSpec(%q) error = %q, want it to name %q", tc.spec, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAgentSpec(%q) unexpected error: %v", tc.spec, err)
			}
			if !reflect.DeepEqual(ids(got), tc.want) {
				t.Errorf("ParseAgentSpec(%q) = %v, want %v", tc.spec, ids(got), tc.want)
			}
		})
	}
}

func TestParseAgentSpecDoesNotAliasRegistry(t *testing.T) {
	got, err := ParseAgentSpec("all")
	if err != nil {
		t.Fatalf("ParseAgentSpec(\"all\") error: %v", err)
	}
	got[0].Instructions[0] = "MUTATED.md"

	second, err := ParseAgentSpec("all")
	if err != nil {
		t.Fatalf("ParseAgentSpec(\"all\") second call error: %v", err)
	}
	if second[0].Instructions[0] != "AGENTS.md" {
		t.Errorf("mutating the parsed slice corrupted a later call: Instructions[0] = %q", second[0].Instructions[0])
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func mustLookup(t *testing.T, id AgentID) Agent {
	t.Helper()
	a, ok := LookupAgent(id)
	if !ok {
		t.Fatalf("LookupAgent(%q) not found", id)
	}
	return a
}

func targetFiles(plan []InstructionTarget) []string {
	out := make([]string, len(plan))
	for i, tgt := range plan {
		out[i] = tgt.File
	}
	return out
}

func findTarget(plan []InstructionTarget, file string) (InstructionTarget, bool) {
	for _, tgt := range plan {
		if tgt.File == file {
			return tgt, true
		}
	}
	return InstructionTarget{}, false
}

// planCase pins every observable of one PlanInstructionTargets call: which
// files are included, whether each is created, and each file's registry-fact
// Agents list.
type planCase struct {
	name       string
	selection  []AgentID
	existsSet  map[string]bool
	explicit   bool
	wantFiles  []string
	wantCreate map[string]bool
	wantAgents map[string][]AgentID
}

// TestPlanInstructionTargetsSelectionCases pins the inclusion matrix from the
// corrected contract: canonical is present except when explicit and unread; any
// other file enters by existence or as the PRIMARY of a named agent; Agents is a
// registry fact in every case. It covers required cases (a)-(h).
func TestPlanInstructionTargetsSelectionCases(t *testing.T) {
	all := []AgentID{AgentPi, AgentCodex, AgentOpenCode, AgentClaude}
	canonical := []AgentID{AgentPi, AgentCodex, AgentOpenCode}
	claudeReaders := []AgentID{AgentPi, AgentClaude}

	cases := []planCase{
		{
			name:       "a auto, CLAUDE.md absent",
			selection:  all,
			existsSet:  nil,
			explicit:   false,
			wantFiles:  []string{"AGENTS.md"},
			wantCreate: map[string]bool{"AGENTS.md": true},
			wantAgents: map[string][]AgentID{"AGENTS.md": canonical},
		},
		{
			name:       "a auto, CLAUDE.md present",
			selection:  all,
			existsSet:  map[string]bool{"CLAUDE.md": true},
			explicit:   false,
			wantFiles:  []string{"AGENTS.md", "CLAUDE.md"},
			wantCreate: map[string]bool{"AGENTS.md": true, "CLAUDE.md": false},
			wantAgents: map[string][]AgentID{"AGENTS.md": canonical, "CLAUDE.md": claudeReaders},
		},
		{
			name:       "b claude, CLAUDE.md absent",
			selection:  []AgentID{AgentClaude},
			existsSet:  nil,
			explicit:   true,
			wantFiles:  []string{"CLAUDE.md"},
			wantCreate: map[string]bool{"CLAUDE.md": true},
			wantAgents: map[string][]AgentID{"CLAUDE.md": claudeReaders},
		},
		{
			name:       "c claude, CLAUDE.md present",
			selection:  []AgentID{AgentClaude},
			existsSet:  map[string]bool{"CLAUDE.md": true},
			explicit:   true,
			wantFiles:  []string{"CLAUDE.md"},
			wantCreate: map[string]bool{"CLAUDE.md": false},
			wantAgents: map[string][]AgentID{"CLAUDE.md": claudeReaders},
		},
		{
			name:       "c claude, AGENTS.md present but unread stays out",
			selection:  []AgentID{AgentClaude},
			existsSet:  map[string]bool{"AGENTS.md": true, "CLAUDE.md": true},
			explicit:   true,
			wantFiles:  []string{"CLAUDE.md"},
			wantCreate: map[string]bool{"CLAUDE.md": false},
			wantAgents: map[string][]AgentID{"CLAUDE.md": claudeReaders},
		},
		{
			name:       "d pi, CLAUDE.md absent",
			selection:  []AgentID{AgentPi},
			existsSet:  nil,
			explicit:   true,
			wantFiles:  []string{"AGENTS.md"},
			wantCreate: map[string]bool{"AGENTS.md": true},
			wantAgents: map[string][]AgentID{"AGENTS.md": canonical},
		},
		{
			name:       "e pi, CLAUDE.md present",
			selection:  []AgentID{AgentPi},
			existsSet:  map[string]bool{"CLAUDE.md": true},
			explicit:   true,
			wantFiles:  []string{"AGENTS.md", "CLAUDE.md"},
			wantCreate: map[string]bool{"AGENTS.md": true, "CLAUDE.md": false},
			wantAgents: map[string][]AgentID{"AGENTS.md": canonical, "CLAUDE.md": claudeReaders},
		},
		{
			name:       "f codex, CLAUDE.md present",
			selection:  []AgentID{AgentCodex},
			existsSet:  map[string]bool{"CLAUDE.md": true},
			explicit:   true,
			wantFiles:  []string{"AGENTS.md", "CLAUDE.md"},
			wantCreate: map[string]bool{"AGENTS.md": true, "CLAUDE.md": false},
			wantAgents: map[string][]AgentID{"AGENTS.md": canonical, "CLAUDE.md": claudeReaders},
		},
		{
			name:       "g codex,claude, CLAUDE.md absent",
			selection:  []AgentID{AgentCodex, AgentClaude},
			existsSet:  nil,
			explicit:   true,
			wantFiles:  []string{"AGENTS.md", "CLAUDE.md"},
			wantCreate: map[string]bool{"AGENTS.md": true, "CLAUDE.md": true},
			wantAgents: map[string][]AgentID{"AGENTS.md": canonical, "CLAUDE.md": claudeReaders},
		},
		{
			name:       "g codex,claude, CLAUDE.md present",
			selection:  []AgentID{AgentCodex, AgentClaude},
			existsSet:  map[string]bool{"CLAUDE.md": true},
			explicit:   true,
			wantFiles:  []string{"AGENTS.md", "CLAUDE.md"},
			wantCreate: map[string]bool{"AGENTS.md": true, "CLAUDE.md": false},
			wantAgents: map[string][]AgentID{"AGENTS.md": canonical, "CLAUDE.md": claudeReaders},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selection := make([]Agent, 0, len(tc.selection))
			for _, id := range tc.selection {
				selection = append(selection, mustLookup(t, id))
			}
			exists := func(file string) bool { return tc.existsSet[file] }

			plan := PlanInstructionTargets(selection, exists, tc.explicit)

			if files := targetFiles(plan); !reflect.DeepEqual(files, tc.wantFiles) {
				t.Fatalf("plan files = %v, want %v", files, tc.wantFiles)
			}
			for _, tgt := range plan {
				if len(tgt.Agents) == 0 {
					t.Errorf("%s Agents is empty, want the registry readers", tgt.File)
				}
				if got, want := tgt.Create, tc.wantCreate[tgt.File]; got != want {
					t.Errorf("%s Create = %v, want %v", tgt.File, got, want)
				}
				if want, ok := tc.wantAgents[tgt.File]; ok {
					if !reflect.DeepEqual(tgt.Agents, want) {
						t.Errorf("%s Agents = %v, want %v", tgt.File, tgt.Agents, want)
					}
				} else {
					t.Errorf("%s has no pinned Agents expectation", tgt.File)
				}
			}
		})
	}
}

func TestPlanInstructionTargetsClaudeAbsentWhenNotExplicit(t *testing.T) {
	exists := func(string) bool { return false }
	plan := PlanInstructionTargets(Agents(), exists, false)

	if files := targetFiles(plan); !reflect.DeepEqual(files, []string{"AGENTS.md"}) {
		t.Errorf("plan files = %v, want [AGENTS.md] only", files)
	}
}

func TestPlanInstructionTargetsClaudePresentWhenFileExists(t *testing.T) {
	exists := func(file string) bool { return file == "AGENTS.md" || file == "CLAUDE.md" }
	plan := PlanInstructionTargets(Agents(), exists, false)

	if files := targetFiles(plan); !reflect.DeepEqual(files, []string{"AGENTS.md", "CLAUDE.md"}) {
		t.Fatalf("plan files = %v, want [AGENTS.md CLAUDE.md]", files)
	}

	claude, ok := findTarget(plan, "CLAUDE.md")
	if !ok {
		t.Fatal("CLAUDE.md missing from plan")
	}
	if claude.Create {
		t.Error("CLAUDE.md Create = true for an existing file")
	}
	if !reflect.DeepEqual(claude.Agents, []AgentID{AgentPi, AgentClaude}) {
		t.Errorf("CLAUDE.md Agents = %v, want [pi claude]", claude.Agents)
	}

	canonical, _ := findTarget(plan, "AGENTS.md")
	if canonical.Create {
		t.Error("AGENTS.md Create = true for an existing file")
	}
}

func TestPlanInstructionTargetsClaudeCreateWhenExplicitAndAbsent(t *testing.T) {
	exists := func(file string) bool { return file == "AGENTS.md" }
	plan := PlanInstructionTargets(Agents(), exists, true)

	if files := targetFiles(plan); !reflect.DeepEqual(files, []string{"AGENTS.md", "CLAUDE.md"}) {
		t.Fatalf("plan files = %v, want [AGENTS.md CLAUDE.md]", files)
	}

	claude, _ := findTarget(plan, "CLAUDE.md")
	if !claude.Create {
		t.Error("CLAUDE.md Create = false, want true when explicit and absent")
	}

	canonical, _ := findTarget(plan, "AGENTS.md")
	if canonical.Create {
		t.Error("AGENTS.md Create = true for an existing file")
	}
}

func TestPlanInstructionTargetsGroupsCanonicalReaders(t *testing.T) {
	exists := func(string) bool { return false }
	plan := PlanInstructionTargets(Agents(), exists, false)

	canonical, ok := findTarget(plan, "AGENTS.md")
	if !ok {
		t.Fatal("AGENTS.md missing from plan")
	}
	if !reflect.DeepEqual(canonical.Agents, []AgentID{AgentPi, AgentCodex, AgentOpenCode}) {
		t.Errorf("AGENTS.md Agents = %v, want [pi codex opencode]", canonical.Agents)
	}
}

func TestPlanInstructionTargetsDeterministicOrder(t *testing.T) {
	// A jumbled input must still resolve to canonical-first, then sorted names,
	// with each target's agent list in registry order (a registry fact, not a
	// projection of the selection).
	agents := []Agent{
		mustLookup(t, AgentClaude),
		mustLookup(t, AgentPi),
		mustLookup(t, AgentOpenCode),
	}
	plan := PlanInstructionTargets(agents, func(string) bool { return true }, false)

	if files := targetFiles(plan); !reflect.DeepEqual(files, []string{"AGENTS.md", "CLAUDE.md"}) {
		t.Fatalf("plan files = %v, want [AGENTS.md CLAUDE.md]", files)
	}

	canonical, _ := findTarget(plan, "AGENTS.md")
	if !reflect.DeepEqual(canonical.Agents, []AgentID{AgentPi, AgentCodex, AgentOpenCode}) {
		t.Errorf("AGENTS.md Agents = %v, want [pi codex opencode]", canonical.Agents)
	}

	claude, _ := findTarget(plan, "CLAUDE.md")
	if !reflect.DeepEqual(claude.Agents, []AgentID{AgentPi, AgentClaude}) {
		t.Errorf("CLAUDE.md Agents = %v, want [pi claude]", claude.Agents)
	}
}
