package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePathInWorkspace_ValidRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	subdir := filepath.Join(workspace, "src")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	safePath, err := ValidatePathInWorkspace("src/main.go", workspace)
	if err != nil {
		t.Errorf("Valid relative path rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func TestValidatePathInWorkspace_ValidAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	testFile := filepath.Join(workspace, "file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	safePath, err := ValidatePathInWorkspace(testFile, workspace)
	if err != nil {
		t.Errorf("Valid absolute path rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func TestValidatePathInWorkspace_TraversalWithDoubleDots(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	_, err := ValidatePathInWorkspace("../../../etc/passwd", workspace)
	if err == nil {
		t.Error("Traversal attack not detected")
	}
}

func TestValidatePathInWorkspace_TraversalWithEncodedDots(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	attempts := []string{
		"%2e%2e/etc/passwd",
		"..%2f../etc/passwd",
		"%252e%252e/etc/passwd",
	}

	for _, attempt := range attempts {
		_, err := ValidatePathInWorkspace(attempt, workspace)
		if err == nil {
			t.Errorf("Encoded traversal not detected: %s", attempt)
		}
	}
}

func TestValidatePathInWorkspace_AbsolutePathOutsideWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	_, err := ValidatePathInWorkspace("/etc/passwd", workspace)
	if err == nil {
		t.Error("Absolute path outside workspace allowed")
	}
}

func TestValidatePathInWorkspace_NestedTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	nested := filepath.Join(workspace, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := ValidatePathInWorkspace("a/b/c/../../../../etc/passwd", workspace)
	if err == nil {
		t.Error("Nested traversal not detected")
	}
}

func TestValidatePathInWorkspace_CurrentDirectoryReference(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	if err := os.WriteFile(filepath.Join(workspace, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	safePath, err := ValidatePathInWorkspace("./file.txt", workspace)
	if err != nil {
		t.Errorf("Current directory reference rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func TestValidatePathInWorkspace_ComplexValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	nested := filepath.Join(workspace, "src", "lib")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "mod.go"), []byte("package lib"), 0644); err != nil {
		t.Fatal(err)
	}

	safePath, err := ValidatePathInWorkspace("src/./lib/mod.go", workspace)
	if err != nil {
		t.Errorf("Complex valid path rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func TestValidatePathInWorkspace_WindowsStyleTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	_, err := ValidatePathInWorkspace("..\\..\\etc\\passwd", workspace)
	if err == nil {
		t.Error("Windows style traversal not detected")
	}
}

func TestValidatePathInWorkspace_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	safePath, err := ValidatePathInWorkspace("", workspace)
	if err != nil {
		t.Errorf("Empty path rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil for empty path")
	}
}

func TestValidatePathInWorkspace_WorkspaceRoot(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	safePath, err := ValidatePathInWorkspace(".", workspace)
	if err != nil {
		t.Errorf("Workspace root rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil for workspace root")
	}
}

func TestValidatePathInWorkspace_DeepNesting(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	deepPath := "a/b/c/d/e/f/g/h/file.txt"
	safePath, err := ValidatePathInWorkspace(deepPath, workspace)
	if err != nil {
		t.Errorf("Deep nesting rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func TestValidatePathInWorkspace_SymlinkEscape(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Running as root, symlink tests may not work as expected")
	}

	tmpDir := t.TempDir()
	outsideDir := t.TempDir()
	workspace := tmpDir

	symlinkPath := filepath.Join(workspace, "escape_link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	_, err := ValidatePathInWorkspace("escape_link/secret.txt", workspace)
	if err == nil {
		t.Error("Symlink escape not detected")
	}
}

func TestValidatePathInWorkspace_SymlinkWithinWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	realDir := filepath.Join(workspace, "real_dir")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(workspace, "link_to_real")
	if err := os.Symlink(realDir, symlinkPath); err != nil {
		t.Skipf("Cannot create symlink: %v", err)
	}

	safePath, err := ValidatePathInWorkspace("link_to_real/file.txt", workspace)
	if err != nil {
		t.Errorf("In-workspace symlink rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func TestValidatePathInWorkspace_DoubleSlash(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	safePath, err := ValidatePathInWorkspace("src//main.go", workspace)
	if err != nil {
		t.Errorf("Double slash path rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func TestValidatePathInWorkspace_MultipleTraversalAttempts(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	attempts := []string{
		"../",
		"..",
		"../workspace",
		"./../",
		"a/../..",
		"a/b/../../..",
		"./a/../../..",
	}

	for _, attempt := range attempts {
		_, err := ValidatePathInWorkspace(attempt, workspace)
		if err == nil {
			t.Errorf("Traversal attempt not detected: %s", attempt)
		}
	}
}

func TestSafePath_Path(t *testing.T) {
	safePath := &SafePath{path: "/workspace/file.txt"}
	if safePath.Path() != "/workspace/file.txt" {
		t.Errorf("Path() returned wrong value: %s", safePath.Path())
	}
}

func TestSafePath_String(t *testing.T) {
	safePath := &SafePath{path: "/workspace/file.txt"}
	if safePath.String() != "/workspace/file.txt" {
		t.Errorf("String() returned wrong value: %s", safePath.String())
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/a/b/../c", "/a/c"},
		{"/a/./b", "/a/b"},
		{"/a/b/./c", "/a/b/c"},
		{"./file.txt", "file.txt"},
	}

	for _, test := range tests {
		result := NormalizePath(test.input)
		if result != test.expected {
			t.Errorf("NormalizePath(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestIsPathInWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	if !IsPathInWorkspace("file.txt", workspace) {
		t.Error("Valid path reported as not in workspace")
	}

	if IsPathInWorkspace("../../../etc/passwd", workspace) {
		t.Error("Traversal path reported as in workspace")
	}
}

func TestValidatePathInWorkspace_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	specialPaths := []string{
		"file with spaces.txt",
		"file\twith\ttabs.txt",
	}

	for _, path := range specialPaths {
		safePath, err := ValidatePathInWorkspace(path, workspace)
		if err != nil {
			t.Errorf("Special character path rejected: %s (%v)", path, err)
		}
		if safePath == nil {
			t.Errorf("SafePath is nil for: %s", path)
		}
	}
}

func TestValidatePathInWorkspace_UnicodePath(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	unicodePaths := []string{
		"ファイル.txt",
		"файл.txt",
		"αρχείο.txt",
	}

	for _, path := range unicodePaths {
		safePath, err := ValidatePathInWorkspace(path, workspace)
		if err != nil {
			t.Errorf("Unicode path rejected: %s (%v)", path, err)
		}
		if safePath == nil {
			t.Errorf("SafePath is nil for: %s", path)
		}
	}
}

func TestValidatePathInWorkspace_VeryLongPath(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := tmpDir

	longPath := ""
	for i := 0; i < 100; i++ {
		longPath += "subdir/"
	}
	longPath += "file.txt"

	safePath, err := ValidatePathInWorkspace(longPath, workspace)
	if err != nil {
		t.Errorf("Very long path rejected: %v", err)
	}
	if safePath == nil {
		t.Error("SafePath is nil")
	}
}

func BenchmarkValidatePathInWorkspace_Safe(b *testing.B) {
	tmpDir := b.TempDir()
	workspace := tmpDir
	path := "src/main.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidatePathInWorkspace(path, workspace)
	}
}

func BenchmarkValidatePathInWorkspace_Traversal(b *testing.B) {
	tmpDir := b.TempDir()
	workspace := tmpDir
	path := "../../../etc/passwd"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidatePathInWorkspace(path, workspace)
	}
}
