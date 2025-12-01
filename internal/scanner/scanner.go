package scanner

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IgnoredFile represents a file that is ignored by .gitignore
type IgnoredFile struct {
	Path       string
	Size       int64
	IsSecret   bool   // likely contains secrets (.env, credentials, etc.)
	Category   string // env, key, config, cache, build, other
}

// ScanResult contains the results of scanning a directory
type ScanResult struct {
	RootPath     string
	IgnoredFiles []IgnoredFile
	TotalSize    int64
	SecretCount  int
}

// Scanner scans directories for gitignored files
type Scanner struct {
	ShowAll     bool     // show all ignored files, not just secrets
	Categories  []string // filter by categories
	ExcludeDeps bool     // exclude node_modules, vendor, etc.
}

// NewScanner creates a new scanner
func NewScanner() *Scanner {
	return &Scanner{
		ShowAll:     false,
		ExcludeDeps: true, // exclude deps by default
	}
}

// Scan scans a directory for gitignored files
func (s *Scanner) Scan(rootPath string) (*ScanResult, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	result := &ScanResult{
		RootPath:     absPath,
		IgnoredFiles: []IgnoredFile{},
	}

	// Check if it's a git repository
	if !isGitRepo(absPath) {
		return result, nil
	}

	// Get list of ignored files using git
	ignoredPaths, err := getGitIgnoredFiles(absPath)
	if err != nil {
		return nil, err
	}

	for _, path := range ignoredPaths {
		// Skip dependency directories if ExcludeDeps is true
		if s.ExcludeDeps && isInDepsDir(path) {
			continue
		}

		fullPath := filepath.Join(absPath, path)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // file might not exist
		}

		if info.IsDir() {
			continue // skip directories
		}

		file := IgnoredFile{
			Path:     path,
			Size:     info.Size(),
			Category: categorizeFile(path),
		}
		file.IsSecret = isSecretFile(path, file.Category)

		if s.ShowAll || file.IsSecret {
			result.IgnoredFiles = append(result.IgnoredFiles, file)
			result.TotalSize += file.Size
			if file.IsSecret {
				result.SecretCount++
			}
		}
	}

	return result, nil
}

// isGitRepo checks if the path is inside a git repository
func isGitRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	return cmd.Run() == nil
}

// getGitIgnoredFiles returns a list of files ignored by .gitignore
func getGitIgnoredFiles(repoPath string) ([]string, error) {
	// Use git status to find ignored files
	cmd := exec.Command("git", "status", "--ignored", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var ignored []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Ignored files start with "!! "
		if strings.HasPrefix(line, "!! ") {
			path := strings.TrimPrefix(line, "!! ")
			// Remove trailing slash for directories
			path = strings.TrimSuffix(path, "/")
			ignored = append(ignored, path)
		}
	}

	return ignored, scanner.Err()
}

// categorizeFile determines the category of a file
func categorizeFile(path string) string {
	name := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	// Environment files
	if strings.HasPrefix(name, ".env") || strings.HasPrefix(name, "env.") {
		return "env"
	}

	// Key/credential files
	keyPatterns := []string{
		"key", "secret", "credential", "token", "password",
		"private", "pem", "p12", "pfx", "keystore",
	}
	for _, pattern := range keyPatterns {
		if strings.Contains(name, pattern) {
			return "key"
		}
	}
	if ext == ".pem" || ext == ".key" || ext == ".p12" || ext == ".pfx" {
		return "key"
	}

	// Config files
	if ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".ini" {
		if strings.Contains(name, "config") || strings.Contains(name, "setting") {
			return "config"
		}
	}

	// Build outputs
	buildDirs := []string{"node_modules", "dist", "build", ".next", "__pycache__", "target", "bin", "obj"}
	for _, dir := range buildDirs {
		if strings.Contains(path, dir+"/") || strings.HasPrefix(path, dir+"/") || path == dir {
			return "build"
		}
	}

	// Cache
	if strings.Contains(path, "cache") || strings.HasPrefix(name, ".") && strings.Contains(name, "cache") {
		return "cache"
	}

	// IDE/Editor
	ideDirs := []string{".idea", ".vscode", ".vs"}
	for _, dir := range ideDirs {
		if strings.HasPrefix(path, dir+"/") || path == dir {
			return "ide"
		}
	}

	return "other"
}

// isSecretFile determines if a file likely contains secrets
func isSecretFile(path string, category string) bool {
	// env and key categories are always considered secrets
	if category == "env" || category == "key" {
		return true
	}

	name := strings.ToLower(filepath.Base(path))

	// Additional secret patterns
	secretPatterns := []string{
		"secret", "credential", "password", "token",
		"auth", "api_key", "apikey", ".npmrc", ".netrc",
		"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519",
	}
	for _, pattern := range secretPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	return false
}

// isInDepsDir checks if a path is inside a dependency directory
func isInDepsDir(path string) bool {
	depsDirs := []string{
		"node_modules/",
		"vendor/",
		".venv/",
		"venv/",
		"__pycache__/",
		".gradle/",
		"target/", // Rust/Maven
		"Pods/",   // iOS
	}
	for _, dir := range depsDirs {
		if strings.HasPrefix(path, dir) || strings.Contains(path, "/"+dir) {
			return true
		}
	}
	return false
}
