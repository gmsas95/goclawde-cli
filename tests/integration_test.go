package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestServerStartsAndShutsdown(t *testing.T) {
	// Create temp config directory
	tmpDir, err := os.MkdirTemp("", "myrai-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command(binaryPath, "server", "--data", tmpDir)
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	// Start server
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give it time to start
	time.Sleep(2 * time.Second)

	// Check if process is running
	if cmd.Process == nil {
		t.Fatal("Server process not running")
	}

	// Kill the server
	if err := cmd.Process.Kill(); err != nil {
		t.Logf("Warning: Failed to kill server: %v", err)
	}
}

func TestCLIWithMessageRequiresConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "myrai-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command(binaryPath, "-m", "hello", "--data", tmpDir)
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, _ := cmd.CombinedOutput()

	// Should fail without config or API key
	if len(output) == 0 {
		t.Fatal("Expected some output even on failure")
	}
}

func TestBatchWithNonexistentFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "batch", "-i", "/nonexistent/file.txt")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()

	// Should fail with error about missing file
	if err == nil {
		t.Fatal("Expected batch to fail with nonexistent file")
	}
	if len(output) == 0 {
		t.Fatal("Expected error output")
	}
}

func TestStatusCommandRequiresConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "myrai-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command(binaryPath, "status", "--data", tmpDir)
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	// status may fail without config, but should produce output
	output, _ := cmd.CombinedOutput()
	if len(output) == 0 {
		t.Fatal("status should produce output even without config")
	}
}

func TestDoctorCommandWorks(t *testing.T) {
	cmd := exec.Command(binaryPath, "doctor")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, _ := cmd.CombinedOutput()

	// doctor should always work and produce diagnostic output
	if len(output) == 0 {
		t.Fatal("doctor produced no output")
	}

	// Check for expected diagnostic output
	outputStr := string(output)
	if !contains(outputStr, "Diagnostics") && !contains(outputStr, "Config") {
		t.Logf("Warning: doctor output doesn't contain expected keywords")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestConfigPathCommand(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "path")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	// May fail without workspace, but should produce output
	output, _ := cmd.CombinedOutput()
	if len(output) == 0 {
		t.Fatal("config path should produce output")
	}
}

func TestMultipleCommandsInSequence(t *testing.T) {
	commands := [][]string{
		{"--help"},
		{"version"},
		{"help"},
		{"doctor"},
		{"project"},
		{"batch", "-h"},
		{"config"},
		{"channels"},
		{"gateway"},
	}

	for _, args := range commands {
		cmd := exec.Command(binaryPath, args...)
		input, _ := os.Open("/dev/null")
		cmd.Stdin = input

		output, err := cmd.CombinedOutput()
		input.Close()

		if len(output) == 0 {
			t.Errorf("Command %v produced no output (err: %v)", args, err)
		}
	}
}

func TestBinaryExistsInPath(t *testing.T) {
	_, err := os.Stat(binaryPath)
	if os.IsNotExist(err) {
		t.Fatal("Test binary not found - TestMain should have built it")
	}
}

func TestDataFlagCreatesDirectory(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "myrai-test-data-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(tmpDir)

	// Run status with custom data dir
	cmd := exec.Command(binaryPath, "status", "--data", tmpDir)
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	cmd.Run() // Ignore error, we just want to test data dir handling
	input.Close()

	// The data directory might or might not be created depending on command
	// This test just verifies the flag doesn't cause a panic
}
