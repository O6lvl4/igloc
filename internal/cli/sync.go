package cli

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/O6lvl4/igloc/internal/config"
	"github.com/spf13/cobra"
)

// Languages to fetch from github/gitignore
var defaultLanguages = []string{
	"Python",
	"Node",
	"Go",
	"Ruby",
	"Java",
	"Rust",
	"Scala",
	"Haskell",
	"Elixir",
	"Dart",
	"Swift",
	"Objective-C",
	"Kotlin",
	"C++",
	"C",
}

// NewSyncCmd creates the sync command
func NewSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync dependency patterns from GitHub gitignore templates",
		Long: `Fetch the latest gitignore patterns from github/gitignore repository
and extract dependency directory patterns.

The patterns are saved to ~/.config/igloc/patterns.yaml and will be
used by the scan command to exclude dependency directories.

Examples:
  igloc sync              # Fetch all supported languages
  igloc sync --list       # List supported languages`,
		RunE: runSync,
	}

	cmd.Flags().Bool("list", false, "List supported languages")

	return cmd
}

func runSync(cmd *cobra.Command, args []string) error {
	listOnly, _ := cmd.Flags().GetBool("list")

	if listOnly {
		fmt.Println("Supported languages:")
		for _, lang := range defaultLanguages {
			fmt.Printf("  - %s\n", lang)
		}
		return nil
	}

	fmt.Println("Fetching patterns from github/gitignore...")

	cfg := &config.PatternsConfig{
		Version:   1,
		UpdatedAt: time.Now(),
		Languages: make(map[string]*config.Language),
	}

	for _, lang := range defaultLanguages {
		fmt.Printf("  Fetching %s...", lang)

		patterns, err := fetchGitignorePatterns(lang)
		if err != nil {
			fmt.Printf(" ✗ (%v)\n", err)
			continue
		}

		if len(patterns) == 0 {
			fmt.Printf(" (no deps patterns)\n")
			continue
		}

		cfg.Languages[strings.ToLower(lang)] = &config.Language{
			Deps: patterns,
		}
		fmt.Printf(" ✓ (%d patterns)\n", len(patterns))
	}

	// Add some common patterns that might not be in gitignore
	addCommonPatterns(cfg)

	if err := config.SavePatterns(cfg); err != nil {
		return fmt.Errorf("failed to save patterns: %w", err)
	}

	path, _ := config.PatternsFilePath()
	fmt.Printf("\nSaved to %s\n", path)
	fmt.Printf("Total: %d patterns across %d languages\n",
		len(cfg.GetAllDepsDirs()), len(cfg.Languages))

	return nil
}

func fetchGitignorePatterns(lang string) ([]string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/github/gitignore/main/%s.gitignore", lang)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return parseGitignoreForDeps(resp.Body)
}

func parseGitignoreForDeps(r io.Reader) ([]string, error) {
	var patterns []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// We're looking for directory patterns that are likely dependencies
		// These typically end with / or are known dependency directories
		pattern := extractDepsPattern(line)
		if pattern != "" && !seen[pattern] {
			seen[pattern] = true
			patterns = append(patterns, pattern)
		}
	}

	return patterns, scanner.Err()
}

func extractDepsPattern(line string) string {
	// Remove negation prefix
	if strings.HasPrefix(line, "!") {
		return ""
	}

	// Remove leading slash (we handle paths relatively)
	line = strings.TrimPrefix(line, "/")

	// Known dependency/build directory keywords
	depsKeywords := []string{
		"node_modules",
		"vendor",
		"venv",
		".venv",
		"env",
		"__pycache__",
		"site-packages",
		"packages",
		".eggs",
		"eggs",
		"dist",
		"build",
		"target",
		"deps",
		"_build",
		".gradle",
		".m2",
		"Pods",
		"Carthage",
		".dart_tool",
		".pub",
		".stack-work",
		"dist-newstyle",
		".cabal",
		"elm-stuff",
		"bower_components",
		".bundle",
		".cargo",
		"pkg",
		"bin",
		"obj",
		"out",
		"lib",
		"libs",
		".nuget",
		".paket",
		"jspm_packages",
		".pnp",
		".yarn",
		"__pypackages__",
		".tox",
		".nox",
		".mypy_cache",
		".pytest_cache",
		".hypothesis",
		".ruff_cache",
		".pixi",
	}

	// Check if line contains any deps keyword
	lineLower := strings.ToLower(line)
	for _, keyword := range depsKeywords {
		if strings.Contains(lineLower, strings.ToLower(keyword)) {
			// Ensure it ends with /
			if !strings.HasSuffix(line, "/") {
				// Only add trailing slash if it looks like a directory
				if !strings.Contains(line, "*") && !strings.Contains(line, ".") {
					line = line + "/"
				} else if strings.HasSuffix(line, "/") {
					// Already has slash
				} else {
					// Skip file patterns
					continue
				}
			}
			return line
		}
	}

	// Also capture any line ending with / that looks like a deps dir
	if strings.HasSuffix(line, "/") {
		name := strings.TrimSuffix(line, "/")
		name = strings.TrimPrefix(name, "**/")

		// Skip if it has wildcards in the middle
		if strings.Contains(name, "*") {
			return ""
		}

		return line
	}

	return ""
}

func addCommonPatterns(cfg *config.PatternsConfig) {
	// Add patterns that might not be in gitignore but are common
	common := &config.Language{
		Deps: []string{
			// General
			".cache/",
			".tmp/",
			"tmp/",
			"temp/",

			// CI/CD artifacts
			".circleci/",
			".github/",
			".gitlab/",

			// IDE caches (not really deps but often ignored)
			".idea/",
			".vscode/",
			".vs/",
		},
	}
	cfg.Languages["common"] = common
}
