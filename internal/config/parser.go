package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {

	configContent, err := os.ReadFile(path)
	if (err) != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	c, err := Parse(configContent)
	if (err) != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return c, nil
}

func Parse(data []byte) (*Config, error) {
	c := &Config{}

	err := yaml.Unmarshal(data, c)
	if (err) != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return c, nil
}
