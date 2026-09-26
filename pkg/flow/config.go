package flow

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the dflow configuration structure,
// typically stored in a .dflow.yaml file at the project root.
type Config struct {
	Branches struct {
		Main     string `yaml:"main"`
		Develop  string `yaml:"develop"`
		Uat      string `yaml:"uat"`
		Features string `yaml:"features"`
		Releases string `yaml:"releases"`
		Hotfixes string `yaml:"hotfixes"`
		Bugfixes string `yaml:"bugfixes"`
	} `yaml:"branches"`
	Flow     FlowConfig     `yaml:"flow"`
	Workflow WorkflowConfig `yaml:"workflow"`
}

type BranchFlowRule struct {
	Base          string   `yaml:"base"`
	FinishTargets []string `yaml:"finish_targets"`
}

type FlowConfig struct {
	Feature BranchFlowRule `yaml:"feature"`
	Release BranchFlowRule `yaml:"release"`
	Hotfix  BranchFlowRule `yaml:"hotfix"`
	Bugfix  BranchFlowRule `yaml:"bugfix"`

	configured map[string]bool
}

type WorkflowBranchRule struct {
	MergeMode string `yaml:"merge_mode"`
}

type WorkflowConfig struct {
	DefaultMergeMode string                        `yaml:"default_merge_mode,omitempty"`
	BranchRules      map[string]WorkflowBranchRule `yaml:"branch_rules"`
}

// Validate checks the complete configuration and reports the first invalid YAML path.
func (c Config) Validate() error {
	branches := []struct{ name, value string }{
		{"branches.main", c.Branches.Main}, {"branches.develop", c.Branches.Develop}, {"branches.uat", c.Branches.Uat},
		{"branches.features", c.Branches.Features}, {"branches.releases", c.Branches.Releases},
		{"branches.hotfixes", c.Branches.Hotfixes}, {"branches.bugfixes", c.Branches.Bugfixes},
	}
	for _, branch := range branches {
		if strings.TrimSpace(branch.value) == "" {
			return fmt.Errorf("%s: must not be blank", branch.name)
		}
	}
	prefixes := branches[3:]
	for i := range prefixes {
		for j := i + 1; j < len(prefixes); j++ {
			if strings.HasPrefix(prefixes[i].value, prefixes[j].value) || strings.HasPrefix(prefixes[j].value, prefixes[i].value) {
				return fmt.Errorf("branches: prefixes %s and %s overlap", prefixes[i].name, prefixes[j].name)
			}
		}
	}
	flows := []struct {
		name       string
		rule       BranchFlowRule
		configured bool
	}{
		{"flow.feature", c.Flow.Feature, c.Flow.configured["feature"]}, {"flow.release", c.Flow.Release, c.Flow.configured["release"]},
		{"flow.hotfix", c.Flow.Hotfix, c.Flow.configured["hotfix"]}, {"flow.bugfix", c.Flow.Bugfix, c.Flow.configured["bugfix"]},
	}
	for _, item := range flows {
		// A zero-valued rule is omitted, including the empty mapping emitted by
		// MarshalYAML for a partially configured fixture. Once either field is
		// present, however, validate the rule as a whole.
		if strings.TrimSpace(item.rule.Base) == "" && len(item.rule.FinishTargets) == 0 {
			continue
		}
		if strings.TrimSpace(item.rule.Base) == "" {
			return fmt.Errorf("%s.base: must not be blank", item.name)
		}
		if len(item.rule.FinishTargets) == 0 {
			return fmt.Errorf("%s.finish_targets: must not be empty", item.name)
		}
		seen := make(map[string]struct{}, len(item.rule.FinishTargets))
		for i, target := range item.rule.FinishTargets {
			path := fmt.Sprintf("%s.finish_targets[%d]", item.name, i)
			if strings.TrimSpace(target) == "" {
				return fmt.Errorf("%s: must not be blank", path)
			}
			if _, ok := seen[target]; ok {
				return fmt.Errorf("%s.finish_targets: duplicate target %q", item.name, target)
			}
			seen[target] = struct{}{}
		}
	}
	if c.Workflow.DefaultMergeMode != "" && !IsValidMergeMode(c.Workflow.DefaultMergeMode) {
		return fmt.Errorf("workflow.default_merge_mode: must be auto or manual")
	}
	for branch, rule := range c.Workflow.BranchRules {
		if strings.TrimSpace(branch) == "" {
			return fmt.Errorf("workflow.branch_rules: branch name must not be blank")
		}
		if !IsValidMergeMode(rule.MergeMode) {
			return fmt.Errorf("workflow.branch_rules.%s.merge_mode: invalid value %q; must be auto or manual", branch, rule.MergeMode)
		}
	}
	return nil
}

func (f *FlowConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("flow: expected a mapping")
	}
	entries := yamlMapping(value)
	decode := func(key string, dst interface{}) error {
		node, ok := entries[key]
		if !ok {
			return nil
		}
		if node.Tag != "!!str" {
			return fmt.Errorf("flow.%s: expected a string", key)
		}
		if err := node.Decode(dst); err != nil {
			return fmt.Errorf("flow.%s: %w", key, err)
		}
		return nil
	}
	nested := make(map[string]BranchFlowRule)
	f.configured = make(map[string]bool)
	for _, name := range []string{"feature", "release", "hotfix", "bugfix"} {
		if node, ok := entries[name]; ok {
			f.configured[name] = true
			var rule BranchFlowRule
			if node.Kind != yaml.MappingNode {
				return fmt.Errorf("flow.%s: expected a mapping", name)
			}
			if err := node.Decode(&rule); err != nil {
				return fmt.Errorf("flow.%s: %w", name, err)
			}
			nested[name] = rule
		}
	}
	legacy := map[string]string{}
	for _, name := range []string{"feature_base", "feature_merge", "release_base", "release_merge", "hotfix_base", "hotfix_merge", "bugfix_base", "bugfix_merge"} {
		var legacyValue string
		if err := decode(name, &legacyValue); err != nil {
			return err
		}
		if _, present := entries[name]; present {
			legacy[name] = legacyValue
			for _, flowName := range []string{"feature", "release", "hotfix", "bugfix"} {
				if strings.HasPrefix(name, flowName+"_") {
					f.configured[flowName] = true
				}
			}
		}
	}
	set := func(name string, rule *BranchFlowRule) error {
		if nestedRule, ok := nested[name]; ok {
			*rule = nestedRule
		}
		baseKey, mergeKey := name+"_base", name+"_merge"
		if value, ok := legacy[baseKey]; ok {
			if rule.Base != "" && rule.Base != value {
				return fmt.Errorf("flow.%s.base: contradicts flow.%s", name, baseKey)
			}
			if rule.Base == "" {
				rule.Base = value
			}
		}
		if value, ok := legacy[mergeKey]; ok {
			if len(rule.FinishTargets) > 0 && (len(rule.FinishTargets) != 1 || rule.FinishTargets[0] != value) {
				return fmt.Errorf("flow.%s.finish_targets: contradicts flow.%s", name, mergeKey)
			}
			if len(rule.FinishTargets) == 0 {
				rule.FinishTargets = []string{value}
			}
		} else if len(rule.FinishTargets) == 0 && nested[name].Base == "" {
			if value, ok := legacy[baseKey]; ok {
				// The original flat schema used *_base as the sole value for
				// release, hotfix, and bugfix finish rules.
				rule.FinishTargets = []string{value}
			}
		}
		return nil
	}
	if err := set("feature", &f.Feature); err != nil {
		return err
	}
	if err := set("release", &f.Release); err != nil {
		return err
	}
	if err := set("hotfix", &f.Hotfix); err != nil {
		return err
	}
	if err := set("bugfix", &f.Bugfix); err != nil {
		return err
	}
	return nil
}

func (f FlowConfig) MarshalYAML() (interface{}, error) {
	return struct {
		Feature BranchFlowRule `yaml:"feature"`
		Release BranchFlowRule `yaml:"release"`
		Hotfix  BranchFlowRule `yaml:"hotfix"`
		Bugfix  BranchFlowRule `yaml:"bugfix"`
	}{f.Feature, f.Release, f.Hotfix, f.Bugfix}, nil
}

func (w *WorkflowConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("workflow: expected a mapping")
	}
	entries := yamlMapping(value)
	if node, ok := entries["default_merge_mode"]; ok {
		if node.Tag != "!!str" {
			return fmt.Errorf("workflow.default_merge_mode: expected a string")
		}
		if err := node.Decode(&w.DefaultMergeMode); err != nil {
			return fmt.Errorf("workflow.default_merge_mode: %w", err)
		}
	}
	w.BranchRules = make(map[string]WorkflowBranchRule)
	node, ok := entries["branch_rules"]
	if !ok {
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("workflow.branch_rules: expected a mapping")
	}
	for i := 0; i < len(node.Content); i += 2 {
		name, keyNode := node.Content[i].Value, node.Content[i]
		value := node.Content[i+1]
		var rule WorkflowBranchRule
		switch value.Kind {
		case yaml.ScalarNode:
			if value.Tag != "!!str" {
				return fmt.Errorf("workflow.branch_rules[%q]: merge mode must be a string", keyNode.Value)
			}
			if err := value.Decode(&rule.MergeMode); err != nil {
				return fmt.Errorf("workflow.branch_rules[%q]: %w", name, err)
			}
		case yaml.MappingNode:
			mergeNode := yamlMapping(value)["merge_mode"]
			if mergeNode == nil {
				return fmt.Errorf("workflow.branch_rules[%q].merge_mode: is required", keyNode.Value)
			}
			if mergeNode.Tag != "!!str" {
				return fmt.Errorf("workflow.branch_rules[%q].merge_mode: must be a string", keyNode.Value)
			}
			if err := value.Decode(&rule); err != nil {
				return fmt.Errorf("workflow.branch_rules[%q]: %w", name, err)
			}
		default:
			return fmt.Errorf("workflow.branch_rules[%q]: expected a string or mapping", keyNode.Value)
		}
		w.BranchRules[name] = rule
	}
	return nil
}

func (w WorkflowConfig) MarshalYAML() (interface{}, error) {
	return struct {
		DefaultMergeMode string                        `yaml:"default_merge_mode,omitempty"`
		BranchRules      map[string]WorkflowBranchRule `yaml:"branch_rules"`
	}(w), nil
}

func GetMergeModeForBranch(cfg *Config, branch string) string {
	if mode, ok := cfg.Workflow.BranchRules[branch]; ok {
		return mode.MergeMode
	}
	return cfg.Workflow.DefaultMergeMode
}

func yamlMapping(node *yaml.Node) map[string]*yaml.Node {
	result := make(map[string]*yaml.Node, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		result[node.Content[i].Value] = node.Content[i+1]
	}
	return result
}

func uniqueStrings(values []string) []string {
	var unique []string
	for _, value := range values {
		if value == "" || slices.Contains(unique, value) {
			continue
		}
		unique = append(unique, value)
	}
	return unique
}
