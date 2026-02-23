package skills

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitHubInstaller handles installation of skills from GitHub
type GitHubInstaller struct {
	loader    *SkillLoader
	skillsDir string
	client    *http.Client
}

// NewGitHubInstaller creates a new GitHub installer
func NewGitHubInstaller(loader *SkillLoader, skillsDir string) *GitHubInstaller {
	return &GitHubInstaller{
		loader:    loader,
		skillsDir: skillsDir,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

// InstallFromGitHub downloads and installs a skill from a GitHub repository
// Supports formats:
//   - github.com/user/repo
//   - github.com/user/repo@v1.2.0
//   - user/repo
//   - user/repo@v1.2.0
func (gi *GitHubInstaller) InstallFromGitHub(repo string) (*RuntimeSkill, error) {
	// Parse repository reference
	owner, name, version, err := parseRepoRef(repo)
	if err != nil {
		return nil, err
	}

	// Construct URLs
	repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, name)

	// Determine download URL based on version
	var downloadURL string
	if version != "" {
		// Download specific release
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.zip", owner, name, version)
	} else {
		// Download latest main branch
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/main.zip", owner, name)
	}

	// Create skill directory
	skillDir := filepath.Join(gi.skillsDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Download the repository
	log.Printf("[GitHub] Downloading from %s...", downloadURL)
	zipPath := filepath.Join(skillDir, "download.zip")
	if err := gi.downloadFile(downloadURL, zipPath); err != nil {
		return nil, fmt.Errorf("failed to download repository: %w", err)
	}
	defer os.Remove(zipPath)

	// Extract the zip
	log.Printf("[GitHub] Extracting...")
	extractDir := filepath.Join(skillDir, "extracted")
	if err := gi.extractZip(zipPath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract repository: %w", err)
	}
	defer os.RemoveAll(extractDir)

	// Find SKILL.md
	skillPath, err := gi.findSkillFile(extractDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find SKILL.md: %w", err)
	}

	// Validate manifest
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	manifest, _, err := ParseSkillMarkdown(string(content))
	if err != nil {
		return nil, fmt.Errorf("invalid SKILL.md: %w", err)
	}

	// Check minimum Myrai version
	if manifest.MinMyraiVersion != "" {
		// TODO: Implement version comparison
		log.Printf("[GitHub] Required Myrai version: %s", manifest.MinMyraiVersion)
	}

	// Move files to final location
	finalDir := filepath.Join(gi.skillsDir, manifest.Name)
	if err := os.RemoveAll(finalDir); err != nil {
		return nil, fmt.Errorf("failed to clean existing directory: %w", err)
	}

	// Copy extracted files to final location
	extractedDir := filepath.Dir(skillPath)
	if err := gi.copyDir(extractedDir, finalDir); err != nil {
		return nil, fmt.Errorf("failed to install skill files: %w", err)
	}

	// Load the skill
	finalSkillPath := filepath.Join(finalDir, "SKILL.md")
	skill, err := gi.loader.LoadSkill(finalSkillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load installed skill: %w", err)
	}

	// Set GitHub-specific metadata
	skill.Source = SourceGitHub
	skill.SourceURL = repoURL
	if version != "" {
		skill.SourceURL = fmt.Sprintf("%s@%s", repoURL, version)
	}

	// Register the skill
	if err := gi.loader.registry.RegisterSkill(skill); err != nil {
		return nil, fmt.Errorf("failed to register skill: %w", err)
	}

	log.Printf("[GitHub] Skill '%s' installed successfully from %s", manifest.Name, repoURL)
	return skill, nil
}

// UpdateSkill updates an existing skill from GitHub
func (gi *GitHubInstaller) UpdateSkill(skillName string) (*RuntimeSkill, error) {
	skill, ok := gi.loader.registry.GetRuntimeSkill(skillName)
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", skillName)
	}

	if skill.Source != SourceGitHub {
		return nil, fmt.Errorf("skill '%s' was not installed from GitHub", skillName)
	}

	// Parse the source URL to get repository info
	repoURL := skill.SourceURL
	if idx := strings.Index(repoURL, "@"); idx != -1 {
		repoURL = repoURL[:idx] // Remove version tag
	}

	// Extract owner/repo from URL
	repoURL = strings.TrimPrefix(repoURL, "https://")
	repoURL = strings.TrimPrefix(repoURL, "github.com/")

	// Uninstall old version
	if err := gi.UninstallSkill(skillName); err != nil {
		return nil, fmt.Errorf("failed to uninstall old version: %w", err)
	}

	// Install new version (latest)
	return gi.InstallFromGitHub(repoURL)
}

// UninstallSkill removes a skill installed from GitHub
func (gi *GitHubInstaller) UninstallSkill(skillName string) error {
	skillDir := filepath.Join(gi.skillsDir, skillName)
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}

	// Unregister from registry
	if err := gi.loader.registry.UnregisterSkill(skillName); err != nil {
		return fmt.Errorf("failed to unregister skill: %w", err)
	}

	log.Printf("[GitHub] Skill '%s' uninstalled", skillName)
	return nil
}

// SearchGitHub searches for skills on GitHub
// This is a simple implementation that would be enhanced with actual GitHub API integration
func (gi *GitHubInstaller) SearchGitHub(query string) ([]GitHubSkillInfo, error) {
	// For now, return a placeholder response
	// In production, this would use the GitHub API to search for repositories
	// with the topic "myrai-skill" or similar

	results := []GitHubSkillInfo{
		{
			Name:        "docker-helper",
			Repo:        "myrai-agents/docker-helper",
			Description: "Docker container management operations",
			Stars:       245,
			Version:     "1.2.0",
		},
		{
			Name:        "kubernetes",
			Repo:        "myrai-agents/kubernetes",
			Description: "Kubernetes cluster management",
			Stars:       189,
			Version:     "2.0.1",
		},
		{
			Name:        "aws-cli",
			Repo:        "myrai-agents/aws-cli",
			Description: "AWS CLI wrapper for common operations",
			Stars:       156,
			Version:     "1.5.0",
		},
	}

	// Filter by query
	var filtered []GitHubSkillInfo
	query = strings.ToLower(query)
	for _, result := range results {
		if strings.Contains(strings.ToLower(result.Name), query) ||
			strings.Contains(strings.ToLower(result.Description), query) {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

// GitHubSkillInfo represents information about a skill on GitHub
type GitHubSkillInfo struct {
	Name        string
	Repo        string
	Description string
	Stars       int
	Version     string
}

// parseRepoRef parses a repository reference
// Supports: owner/repo, owner/repo@v1.0.0, github.com/owner/repo, github.com/owner/repo@v1.0.0
func parseRepoRef(ref string) (owner, name, version string, err error) {
	// Remove github.com prefix if present
	ref = strings.TrimPrefix(ref, "github.com/")

	// Check for version tag
	if idx := strings.Index(ref, "@"); idx != -1 {
		version = ref[idx+1:]
		ref = ref[:idx]
	}

	// Split owner and name
	parts := strings.Split(ref, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid repository format: %s (expected owner/repo)", ref)
	}

	owner = parts[0]
	name = parts[1]

	// Validate
	if owner == "" || name == "" {
		return "", "", "", fmt.Errorf("invalid repository format: owner and name cannot be empty")
	}

	// Validate version format if provided
	if version != "" && !isValidVersion(version) {
		return "", "", "", fmt.Errorf("invalid version format: %s", version)
	}

	return owner, name, version, nil
}

// downloadFile downloads a file from URL to local path
func (gi *GitHubInstaller) downloadFile(url, filepath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Add headers to handle redirects
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s (status %d)", url, resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractZip extracts a zip file to a directory
func (gi *GitHubInstaller) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Security: prevent directory traversal
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// findSkillFile finds the SKILL.md file in the extracted directory
func (gi *GitHubInstaller) findSkillFile(dir string) (string, error) {
	var skillPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "SKILL.md" {
			skillPath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if skillPath == "" {
		return "", fmt.Errorf("SKILL.md not found in repository")
	}

	return skillPath, nil
}

// copyDir copies a directory recursively
func (gi *GitHubInstaller) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return gi.copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func (gi *GitHubInstaller) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	// Copy permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, info.Mode())
}
