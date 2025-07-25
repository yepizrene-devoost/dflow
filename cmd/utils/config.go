// Package utils provides shared utility functions used throughout the dflow CLI.
//
// It includes helpers for:
//
//   - Reading and writing the .dflow.yaml configuration file
//   - Determining merge behavior based on branch rules
//   - Console output formatting (e.g., colors, emojis, status messages)
//   - Signal handling for graceful termination (e.g., Ctrl+C)
package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the dflow configuration structure,
// typically stored in a .dflow.yaml file at the project root.
//
// It includes branching information, flow rules, and merge behavior.
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

	Flow struct {
		FeatureBase  string `yaml:"feature_base"`
		FeatureMerge string `yaml:"feature_merge"`
		ReleaseBase  string `yaml:"release_base"`
		HotfixBase   string `yaml:"hotfix_base"`
		BugfixBase   string `yaml:"bugfix_base"`
	} `yaml:"flow"`

	Workflow struct {
		DefaultMergeMode string            `yaml:"default_merge_mode"`
		BranchRules      map[string]string `yaml:"branch_rules"` // e.g., {"main": "manual", "develop": "auto"}
	} `yaml:"workflow"`
}

const bannerToConfig = `
#
#               ██████╗ ███████╗██╗      ██████╗ ██╗    ██╗
#               ██╔══██╗██╔════╝██║     ██╔═══██╗██║    ██║
#               ██║  ██║█████╗  ██║     ██║   ██║██║ █╗ ██║
#               ██║  ██║██╔══╝  ██║     ██║   ██║██║███╗██║
#               ██████╔╝██║     ███████╗╚██████╔╝╚███╔███╔╝
#               ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ 

#            dflow config file - autogenerated by 'dflow init'
`

// LoadConfig reads and parses the .dflow.yaml configuration file
// from the current working directory (or DFLOW_CWD if set).
// Returns a populated Config struct or an error.
func LoadConfig() (*Config, error) {
	dir := os.Getenv("DFLOW_CWD")
	if dir == "" {
		dir, _ = os.Getwd()
	}

	path := dir + string(os.PathSeparator) + ".dflow.yaml"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read .dflow.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing .dflow.yaml: %v", err)
	}

	return &cfg, nil
}

// SaveConfig writes the given Config struct to a .dflow.yaml file
// in the current working directory (or DFLOW_CWD if set).
//
// The file will be overwritten if it already exists.
// A banner header is included for identification.
func SaveConfig(cfg *Config) error {
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error generating YAML: %v", err)
	}

	finalContent := []byte(bannerToConfig + "\n" + string(yamlData))

	dir := os.Getenv("DFLOW_CWD")
	if dir == "" {
		dir, _ = os.Getwd()
	}

	path := dir + string(os.PathSeparator) + ".dflow.yaml"

	if err := os.WriteFile(path, finalContent, 0644); err != nil {
		return fmt.Errorf("error writing .dflow.yaml: %v", err)
	}

	return nil
}

// GetMergeModeForBranch returns the merge mode ("auto" or "manual")
// for the given branch, based on the .dflow.yaml configuration.
//
// This is used by dflow to decide whether to create a Pull Request or
// merge directly from the CLI, depending on branch-specific rules.
func GetMergeModeForBranch(cfg *Config, branch string) string {
	if mode, ok := cfg.Workflow.BranchRules[branch]; ok {
		return mode
	}
	return cfg.Workflow.DefaultMergeMode
}
