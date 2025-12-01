package cli

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/O6lvl4/igloc/internal/config"
	"github.com/O6lvl4/igloc/internal/scanner"
	"gopkg.in/yaml.v3"
	"github.com/spf13/cobra"
)

// Manifest describes the contents of an export archive
type Manifest struct {
	Version   int           `yaml:"version"`
	CreatedAt time.Time     `yaml:"created_at"`
	Machine   string        `yaml:"machine,omitempty"`
	Repos     []RepoExport  `yaml:"repos"`
}

// RepoExport describes exported files from a repository
type RepoExport struct {
	Name   string   `yaml:"name"`
	Path   string   `yaml:"path"`
	Files  []string `yaml:"files"`
}

var (
	exportRecursive   bool
	exportIncludeDeps bool
)

// NewExportCmd creates the export command
func NewExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [output.zip]",
		Short: "Export ignored files to a zip archive",
		Long: `Export gitignored files (secrets, configs) to a zip archive.

This creates a portable backup of all your secret files that can be
imported on another machine.

Examples:
  igloc export backup.zip                    # Export current repo
  igloc export -r ~/projects secrets.zip     # Export all repos recursively
  igloc export --path ~/myapp backup.zip     # Export specific directory`,
		Args: cobra.ExactArgs(1),
		RunE: runExport,
	}

	cmd.Flags().BoolVarP(&exportRecursive, "recursive", "r", false, "Recursively scan subdirectories for git repos")
	cmd.Flags().String("path", ".", "Path to scan")
	cmd.Flags().BoolVar(&exportIncludeDeps, "include-deps", false, "Include files in node_modules, vendor, etc.")

	return cmd
}

func runExport(cmd *cobra.Command, args []string) error {
	outputPath := args[0]
	scanPath, _ := cmd.Flags().GetString("path")

	absPath, err := filepath.Abs(scanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	fmt.Printf("Scanning %s...\n", absPath)

	// Collect files to export
	var repos []RepoExport
	s := scanner.NewScanner()
	s.ExcludeDeps = !exportIncludeDeps

	if exportRecursive {
		repos, err = collectReposRecursive(s, absPath)
	} else {
		repos, err = collectSingleRepo(s, absPath)
	}
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		fmt.Println("No files to export.")
		return nil
	}

	// Count total files
	totalFiles := 0
	for _, repo := range repos {
		totalFiles += len(repo.Files)
	}

	fmt.Printf("Found %d files in %d repositories\n", totalFiles, len(repos))

	// Create zip file
	if err := createExportZip(outputPath, repos); err != nil {
		return fmt.Errorf("failed to create zip: %w", err)
	}

	// Get file size
	info, _ := os.Stat(outputPath)
	fmt.Printf("\nExported to %s (%s)\n", outputPath, formatSize(info.Size()))

	return nil
}

func collectSingleRepo(s *scanner.Scanner, path string) ([]RepoExport, error) {
	result, err := s.Scan(path)
	if err != nil {
		return nil, err
	}

	if len(result.IgnoredFiles) == 0 {
		return nil, nil
	}

	var files []string
	for _, f := range result.IgnoredFiles {
		if f.IsSecret {
			files = append(files, f.Path)
		}
	}

	if len(files) == 0 {
		return nil, nil
	}

	repoName := filepath.Base(path)
	return []RepoExport{
		{
			Name:  repoName,
			Path:  path,
			Files: files,
		},
	}, nil
}

func collectReposRecursive(s *scanner.Scanner, rootPath string) ([]RepoExport, error) {
	var repos []RepoExport

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			repoExports, err := collectSingleRepo(s, repoPath)
			if err == nil && len(repoExports) > 0 {
				repos = append(repos, repoExports...)
			}
			return filepath.SkipDir
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

	return repos, err
}

func createExportZip(outputPath string, repos []RepoExport) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Create manifest
	hostname, _ := os.Hostname()
	manifest := Manifest{
		Version:   1,
		CreatedAt: time.Now(),
		Machine:   hostname,
		Repos:     repos,
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}

	// Write manifest
	manifestWriter, err := zipWriter.Create("manifest.yaml")
	if err != nil {
		return err
	}
	manifestWriter.Write(manifestData)

	// Write patterns.yaml if it exists
	patternsPath, _ := config.PatternsFilePath()
	if patternsData, err := os.ReadFile(patternsPath); err == nil {
		patternsWriter, err := zipWriter.Create("patterns.yaml")
		if err != nil {
			return err
		}
		patternsWriter.Write(patternsData)
	}

	// Write files
	for _, repo := range repos {
		fmt.Printf("  Exporting %s (%d files)\n", repo.Name, len(repo.Files))

		for _, filePath := range repo.Files {
			fullPath := filepath.Join(repo.Path, filePath)
			zipPath := filepath.Join("files", repo.Name, filePath)

			if err := addFileToZip(zipWriter, fullPath, zipPath); err != nil {
				fmt.Printf("    Warning: could not add %s: %v\n", filePath, err)
				continue
			}
		}
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, sourcePath, zipPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}
