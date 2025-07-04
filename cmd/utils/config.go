package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Branches struct {
		Main     string `yaml:"main"`
		Develop  string `yaml:"develop"`
		Uat      string `yaml:"uat"`
		Features string `yaml:"features"`
		Releases string `yaml:"releases"`
		Hotfixes string `yaml:"hotfixes"`
	} `yaml:"branches"`

	Flow struct {
		FeatureBase  string `yaml:"feature_base"`
		FeatureMerge string `yaml:"feature_merge"`
		ReleaseBase  string `yaml:"release_base"`
		HotfixBase   string `yaml:"hotfix_base"`
	} `yaml:"flow"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(".dflow.yaml")
	if err != nil {
		return nil, fmt.Errorf(".dflow.yaml not found. Run `dflow init` first")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing .dflow.yaml: %v", err)
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error generating YAML: %v", err)
	}
	if err := os.WriteFile(".dflow.yaml", data, 0644); err != nil {
		return fmt.Errorf("error writing .dflow.yaml: %v", err)
	}
	return nil
}
