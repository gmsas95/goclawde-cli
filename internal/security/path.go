package security

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrPathTraversal         = errors.New("path traversal detected")
	ErrPathOutsideWorkspace  = errors.New("path escapes workspace")
	ErrSymlinkEscape         = errors.New("symlink escape detected")
	ErrInvalidPath           = errors.New("invalid path")
)

var traversalPatterns = []string{
	"..",
	"%2e%2e",
	"%252e%252e",
	"..%2f",
	"%2f..",
	"..\\",
	"\\..\\",
}

type SafePath struct {
	path string
}

func (sp *SafePath) Path() string {
	return sp.path
}

func (sp *SafePath) String() string {
	return sp.path
}

func ValidatePathInWorkspace(path, workspace string) (*SafePath, error) {
	if containsTraversalPattern(path) {
		return nil, ErrPathTraversal
	}

	workspacePath := filepath.Clean(workspace)
	if !filepath.IsAbs(workspacePath) {
		absW, err := filepath.Abs(workspacePath)
		if err != nil {
			return nil, ErrInvalidPath
		}
		workspacePath = absW
	}

	var targetPath string
	if filepath.IsAbs(path) {
		targetPath = filepath.Clean(path)
	} else {
		targetPath = filepath.Join(workspacePath, path)
	}

	targetPath = filepath.Clean(targetPath)

	if err := checkSymlinkEscape(targetPath, workspacePath); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(targetPath, workspacePath+string(os.PathSeparator)) && targetPath != workspacePath {
		return nil, ErrPathOutsideWorkspace
	}

	return &SafePath{path: targetPath}, nil
}

func containsTraversalPattern(path string) bool {
	lowerPath := strings.ToLower(path)
	for _, pattern := range traversalPatterns {
		if strings.Contains(lowerPath, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func checkSymlinkEscape(targetPath, workspacePath string) error {
	relPath, err := filepath.Rel(workspacePath, targetPath)
	if err != nil {
		return nil
	}

	if strings.HasPrefix(relPath, "..") {
		return ErrPathOutsideWorkspace
	}

	currentPath := workspacePath
	parts := strings.Split(relPath, string(os.PathSeparator))

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		currentPath = filepath.Join(currentPath, part)

		info, err := os.Lstat(currentPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ErrInvalidPath
		}

		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(currentPath)
			if err != nil {
				continue
			}

			resolved = filepath.Clean(resolved)
			if !strings.HasPrefix(resolved, workspacePath+string(os.PathSeparator)) && resolved != workspacePath {
				return ErrSymlinkEscape
			}
		}
	}

	return nil
}

func NormalizePath(path string) string {
	return filepath.Clean(path)
}

func IsPathInWorkspace(path, workspace string) bool {
	_, err := ValidatePathInWorkspace(path, workspace)
	return err == nil
}
