package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// PatternsConfig holds dependency directory patterns
type PatternsConfig struct {
	Version   int                  `yaml:"version"`
	UpdatedAt time.Time            `yaml:"updated_at"`
	Languages map[string]*Language `yaml:"languages"`
}

// Language holds patterns for a specific language
type Language struct {
	Deps []string `yaml:"deps"`
}

// DefaultConfigDir returns the default config directory
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "igloc"), nil
}

// PatternsFilePath returns the path to patterns.yaml
func PatternsFilePath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "patterns.yaml"), nil
}

// LoadPatterns loads patterns from the config file
func LoadPatterns() (*PatternsConfig, error) {
	path, err := PatternsFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config file yet
		}
		return nil, err
	}

	var config PatternsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SavePatterns saves patterns to the config file
func SavePatterns(config *PatternsConfig) error {
	dir, err := DefaultConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := PatternsFilePath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetAllDepsDirs returns all dependency directories from config
func (c *PatternsConfig) GetAllDepsDirs() []string {
	if c == nil || c.Languages == nil {
		return nil
	}

	seen := make(map[string]bool)
	var result []string

	for _, lang := range c.Languages {
		for _, dep := range lang.Deps {
			if !seen[dep] {
				seen[dep] = true
				result = append(result, dep)
			}
		}
	}

	return result
}
