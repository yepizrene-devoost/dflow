package agent

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/pkg/flow"
)

var update = flag.Bool("update", false, "update golden files")

func testConfig() *flow.Config {
	cfg := &flow.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"

	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = []string{"develop"}
	cfg.Flow.Release.Base = "develop"
	cfg.Flow.Release.FinishTargets = []string{"main", "develop"}
	cfg.Flow.Hotfix.Base = "main"
	cfg.Flow.Hotfix.FinishTargets = []string{"main", "develop"}
	cfg.Flow.Bugfix.Base = "develop"
	cfg.Flow.Bugfix.FinishTargets = []string{"develop"}

	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
		"main":    {MergeMode: "manual"},
	}

	return cfg
}

func TestGenerateAgentDocGolden(t *testing.T) {
	doc := GenerateAgentDoc(testConfig())

	golden := "testdata/agent_doc.md"
	if *update {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, doc, 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated golden file %s", golden)
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden file: %v (run with -update to create)", err)
	}

	if !bytes.Equal(doc, want) {
		t.Errorf("GenerateAgentDoc() differs from golden file\ngot:\n%s\nwant:\n%s", doc, want)
	}
}

func TestGenerateAgentDocContainsBranchTypes(t *testing.T) {
	doc := string(GenerateAgentDoc(testConfig()))

	cases := []struct {
		branchType string
		prefix     string
		base       string
	}{
		{"feature", "feature/", "develop"},
		{"release", "release/", "develop"},
		{"hotfix", "hotfix/", "main"},
		{"bugfix", "bugfix/", "develop"},
	}

	for _, tc := range cases {
		if !strings.Contains(doc, tc.branchType) {
			t.Errorf("output missing branch type %q", tc.branchType)
		}
		if !strings.Contains(doc, tc.prefix) {
			t.Errorf("output missing prefix %q for %s", tc.prefix, tc.branchType)
		}
		if !strings.Contains(doc, tc.base) {
			t.Errorf("output missing base %q for %s", tc.base, tc.branchType)
		}
	}
}

func TestGenerateAgentDocContainsMergeRules(t *testing.T) {
	doc := string(GenerateAgentDoc(testConfig()))

	if !strings.Contains(doc, "develop") {
		t.Error("output missing develop branch in merge rules")
	}
	if !strings.Contains(doc, "auto") {
		t.Error("output missing auto merge mode")
	}
	if !strings.Contains(doc, "manual") {
		t.Error("output missing manual merge mode")
	}
	if !strings.Contains(doc, "Default mode: manual") {
		t.Error("output missing default merge mode declaration")
	}
}

func TestGenerateAgentDocContainsStaticSections(t *testing.T) {
	doc := string(GenerateAgentDoc(testConfig()))

	sections := []string{
		"## Commands",
		"## Merge mode is the decision authority",
		"## finish guardrails",
		"## Issue lifecycle",
		"## Chained branches",
		"## JSON output is a contract",
		"## Commit conventions",
	}

	for _, section := range sections {
		if !strings.Contains(doc, section) {
			t.Errorf("output missing section %q", section)
		}
	}
}

func TestGenerateAgentDocContainsFrontmatter(t *testing.T) {
	doc := string(GenerateAgentDoc(testConfig()))

	if !strings.HasPrefix(doc, "---\n") {
		t.Error("output does not start with YAML frontmatter")
	}
	if !strings.Contains(doc, "name: dflow") {
		t.Error("output missing name in frontmatter")
	}
	if !strings.Contains(doc, "description:") {
		t.Error("output missing description in frontmatter")
	}
	if !strings.Contains(doc, "generated_by: dflow agent") {
		t.Error("output missing generated_by metadata")
	}
}

func TestGenerateAgentDocDefaultBranch(t *testing.T) {
	doc := string(GenerateAgentDoc(testConfig()))

	// DefaultBranch should be develop (cfg.Branches.Develop)
	if !strings.Contains(doc, "The repository default branch is `develop`") {
		t.Error("output missing default branch reference to develop")
	}
	if !strings.Contains(doc, "the merge commit on `develop`") {
		t.Error("output missing closing comment reference to develop")
	}
}

func TestGenerateAgentDocIdempotent(t *testing.T) {
	cfg := testConfig()
	doc1 := GenerateAgentDoc(cfg)
	doc2 := GenerateAgentDoc(cfg)

	if !bytes.Equal(doc1, doc2) {
		t.Error("GenerateAgentDoc is not idempotent: two calls produced different output")
	}
}

func TestMergeRuleRowsSorted(t *testing.T) {
	cfg := testConfig()
	rows := mergeRuleRows(cfg)

	if len(rows) != 2 {
		t.Fatalf("expected 2 merge rule rows, got %d", len(rows))
	}

	// Sorted by branch name: develop, main
	if rows[0].Branch != "develop" {
		t.Errorf("first row branch = %q, want develop", rows[0].Branch)
	}
	if rows[1].Branch != "main" {
		t.Errorf("second row branch = %q, want main", rows[1].Branch)
	}
}
