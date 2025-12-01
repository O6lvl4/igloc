package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/O6lvl4/igloc/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	flagAll         bool
	flagRecursive   bool
	flagCategory    string
	flagIncludeDeps bool
)

// NewScanCmd creates the scan command
func NewScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a directory for gitignored files",
		Long: `Scan a directory for files that are ignored by .gitignore.

By default, only files that likely contain secrets are shown (like .env files).
Use --all to see all ignored files.

Examples:
  igloc scan                    # Scan current directory for secrets
  igloc scan ~/projects         # Scan specific directory
  igloc scan -r ~/projects      # Recursively scan all git repos
  igloc scan --all              # Show all ignored files, not just secrets
  igloc scan --category env     # Show only .env files`,
		RunE: runScan,
	}

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "Show all ignored files, not just secrets")
	cmd.Flags().BoolVarP(&flagRecursive, "recursive", "r", false, "Recursively scan subdirectories for git repos")
	cmd.Flags().StringVarP(&flagCategory, "category", "c", "", "Filter by category (env, key, config, build, cache, ide, other)")
	cmd.Flags().BoolVar(&flagIncludeDeps, "include-deps", false, "Include files in node_modules, vendor, etc.")

	return cmd
}

func runScan(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	s := scanner.NewScanner()
	s.ShowAll = flagAll
	s.ExcludeDeps = !flagIncludeDeps

	if flagRecursive {
		return scanRecursive(s, absPath)
	}

	return scanSingle(s, absPath)
}

func scanSingle(s *scanner.Scanner, path string) error {
	result, err := s.Scan(path)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	printResult(result)
	return nil
}

func scanRecursive(s *scanner.Scanner, rootPath string) error {
	var allResults []*scanner.ScanResult
	var totalSecrets int
	var totalFiles int

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Check for .git directory
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			result, err := s.Scan(repoPath)
			if err == nil && len(result.IgnoredFiles) > 0 {
				allResults = append(allResults, result)
				totalSecrets += result.SecretCount
				totalFiles += len(result.IgnoredFiles)
			}
			return filepath.SkipDir // don't descend into .git
		}

		// Skip common non-repo directories
		if info.IsDir() {
			skipDirs := []string{"node_modules", "vendor", ".cache", "__pycache__"}
			for _, skip := range skipDirs {
				if info.Name() == skip {
					return filepath.SkipDir
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(allResults) == 0 {
		fmt.Println("No git repositories found with ignored files.")
		return nil
	}

	// Print results for each repo
	for _, result := range allResults {
		printResult(result)
		fmt.Println()
	}

	// Print summary
	fmt.Println("========================================")
	fmt.Printf("Summary: %d repositories, %d files", len(allResults), totalFiles)
	if totalSecrets > 0 {
		fmt.Printf(", %d secrets", totalSecrets)
	}
	fmt.Println()

	return nil
}

func printResult(result *scanner.ScanResult) {
	if len(result.IgnoredFiles) == 0 {
		fmt.Printf("📂 %s\n", result.RootPath)
		fmt.Println("   No ignored files found.")
		return
	}

	// Filter by category if specified
	files := result.IgnoredFiles
	if flagCategory != "" {
		var filtered []scanner.IgnoredFile
		for _, f := range files {
			if f.Category == flagCategory {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	if len(files) == 0 {
		fmt.Printf("📂 %s\n", result.RootPath)
		fmt.Printf("   No files found in category: %s\n", flagCategory)
		return
	}

	// Group by category
	byCategory := make(map[string][]scanner.IgnoredFile)
	for _, f := range files {
		byCategory[f.Category] = append(byCategory[f.Category], f)
	}

	// Sort categories
	var categories []string
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	fmt.Printf("📂 %s\n", result.RootPath)

	for _, cat := range categories {
		catFiles := byCategory[cat]
		icon := getCategoryIcon(cat)
		fmt.Printf("\n   %s %s (%d)\n", icon, strings.ToUpper(cat), len(catFiles))

		for _, f := range catFiles {
			secretMark := ""
			if f.IsSecret {
				secretMark = " 🔐"
			}
			fmt.Printf("      %s%s\n", f.Path, secretMark)
		}
	}

	fmt.Printf("\n   Total: %d files", len(files))
	if result.SecretCount > 0 {
		fmt.Printf(" (🔐 %d secrets)", result.SecretCount)
	}
	fmt.Println()
}

func getCategoryIcon(category string) string {
	icons := map[string]string{
		"env":    "🔑",
		"key":    "🔐",
		"config": "⚙️",
		"build":  "📦",
		"cache":  "💾",
		"ide":    "🖥️",
		"other":  "📄",
	}
	if icon, ok := icons[category]; ok {
		return icon
	}
	return "📄"
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
