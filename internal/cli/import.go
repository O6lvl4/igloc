package cli

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/O6lvl4/igloc/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	importYes     bool
	importDryRun  bool
	importBaseDir string
)

// NewImportCmd creates the import command
func NewImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [archive.zip]",
		Short: "Import ignored files from a zip archive",
		Long: `Import gitignored files (secrets, configs) from a zip archive.

This restores secret files that were exported from another machine.
By default, it will ask for confirmation before overwriting files.

Examples:
  igloc import backup.zip              # Import with confirmation
  igloc import --yes backup.zip        # Import without confirmation
  igloc import --dry-run backup.zip    # Show what would be imported
  igloc import --base ~/projects backup.zip  # Specify base directory`,
		Args: cobra.ExactArgs(1),
		RunE: runImport,
	}

	cmd.Flags().BoolVarP(&importYes, "yes", "y", false, "Import without confirmation")
	cmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Show what would be imported without actually importing")
	cmd.Flags().StringVar(&importBaseDir, "base", "", "Base directory for imports (default: original paths or current directory)")

	return cmd
}

func runImport(cmd *cobra.Command, args []string) error {
	archivePath := args[0]

	// Open zip file
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer reader.Close()

	// Read manifest
	manifest, err := readManifest(reader)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	fmt.Printf("Archive created: %s\n", manifest.CreatedAt.Format("2006-01-02 15:04:05"))
	if manifest.Machine != "" {
		fmt.Printf("Source machine: %s\n", manifest.Machine)
	}
	fmt.Printf("Repositories: %d\n", len(manifest.Repos))
	fmt.Println()

	// Show what will be imported
	totalFiles := 0
	for _, repo := range manifest.Repos {
		fmt.Printf("📂 %s\n", repo.Name)
		for _, file := range repo.Files {
			destPath := resolveDestPath(repo, file)
			exists := fileExists(destPath)
			status := ""
			if exists {
				status = " (overwrite)"
			}
			fmt.Printf("   %s%s\n", file, status)
			totalFiles++
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d files\n\n", totalFiles)

	if importDryRun {
		fmt.Println("Dry run - no files were imported.")
		return nil
	}

	// Confirm
	if !importYes {
		fmt.Print("Proceed with import? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Import cancelled.")
			return nil
		}
	}

	// Import files
	if err := importFiles(reader, manifest); err != nil {
		return err
	}

	// Import patterns.yaml
	if err := importPatterns(reader); err != nil {
		fmt.Printf("Warning: could not import patterns: %v\n", err)
	}

	fmt.Println("\nImport complete!")
	return nil
}

func readManifest(reader *zip.ReadCloser) (*Manifest, error) {
	for _, file := range reader.File {
		if file.Name == "manifest.yaml" {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			var manifest Manifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				return nil, err
			}

			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("manifest.yaml not found in archive")
}

func resolveDestPath(repo RepoExport, filePath string) string {
	if importBaseDir != "" {
		// Use base directory + repo name + file path
		return filepath.Join(importBaseDir, repo.Name, filePath)
	}

	// Check if original path exists
	if dirExists(repo.Path) {
		return filepath.Join(repo.Path, filePath)
	}

	// Fall back to current directory + repo name
	return filepath.Join(".", repo.Name, filePath)
}

func importFiles(zipReader *zip.ReadCloser, manifest *Manifest) error {
	// Build a map of repo names to their export info
	repoMap := make(map[string]RepoExport)
	for _, repo := range manifest.Repos {
		repoMap[repo.Name] = repo
	}

	for _, file := range zipReader.File {
		// Only process files in the files/ directory
		if !strings.HasPrefix(file.Name, "files/") {
			continue
		}

		// Parse path: files/repoName/path/to/file
		relPath := strings.TrimPrefix(file.Name, "files/")
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) < 2 {
			continue
		}

		repoName := parts[0]
		filePath := parts[1]

		repo, ok := repoMap[repoName]
		if !ok {
			continue
		}

		destPath := resolveDestPath(repo, filePath)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			fmt.Printf("  ✗ %s: %v\n", filePath, err)
			continue
		}

		// Extract file
		if err := extractFile(file, destPath); err != nil {
			fmt.Printf("  ✗ %s: %v\n", filePath, err)
			continue
		}

		fmt.Printf("  ✓ %s\n", destPath)
	}

	return nil
}

func extractFile(zipFile *zip.File, destPath string) error {
	rc, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Preserve original permissions
	if err := outFile.Chmod(zipFile.Mode()); err != nil {
		// Non-fatal, continue
	}

	_, err = io.Copy(outFile, rc)
	return err
}

func importPatterns(zipReader *zip.ReadCloser) error {
	for _, file := range zipReader.File {
		if file.Name == "patterns.yaml" {
			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return err
			}

			// Ensure config directory exists
			configDir, err := config.DefaultConfigDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return err
			}

			patternsPath, err := config.PatternsFilePath()
			if err != nil {
				return err
			}

			return os.WriteFile(patternsPath, data, 0644)
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
