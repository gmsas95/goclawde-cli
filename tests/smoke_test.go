package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Get the project root directory (parent of tests/)
	projectRoot, err := filepath.Abs("..")
	if err != nil {
		panic("Failed to get project root: " + err.Error())
	}

	// Create bin directory if it doesn't exist
	binDir := filepath.Join(projectRoot, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		panic("Failed to create bin directory: " + err.Error())
	}

	binaryPath = filepath.Join(binDir, "myrai_test")

	// Build the binary once
	cmd := exec.Command("go", "build", "-o", binaryPath, filepath.Join(projectRoot, "cmd", "myrai"))
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic("Failed to build test binary: " + err.Error() + "\n" + string(output))
	}

	exitCode := m.Run()

	// Cleanup
	os.Remove(binaryPath)
	os.Exit(exitCode)
}

func TestBinaryHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "--help")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("--help produced no output")
	}
}

func TestBinaryVersion(t *testing.T) {
	cmd := exec.Command(binaryPath, "version")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("version produced no output")
	}
}

func TestBinaryHelpSubcommand(t *testing.T) {
	cmd := exec.Command(binaryPath, "help")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("help produced no output")
	}
}

func TestBinaryDoctor(t *testing.T) {
	cmd := exec.Command(binaryPath, "doctor")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, _ := cmd.CombinedOutput()
	if len(output) == 0 {
		t.Fatal("doctor produced no output")
	}
}

func TestBinaryProjectHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "project")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("project produced no output")
	}
}

func TestBinaryBatchHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "batch", "-h")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("batch -h failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("batch -h produced no output")
	}
}

func TestBinaryConfigHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "config")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("config failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("config produced no output")
	}
}

func TestBinarySkillsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "skills")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	// skills requires config, so it may fail, but should produce output
	output, _ := cmd.CombinedOutput()
	if len(output) == 0 {
		t.Fatal("skills produced no output")
	}
}

func TestBinaryChannelsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "channels")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("channels failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("channels produced no output")
	}
}

func TestBinaryGatewayHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "gateway", "-h")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gateway -h failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("gateway -h produced no output")
	}
}

func TestBinaryPersonaHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "persona")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	// persona requires config, so it may fail, but should produce output
	output, _ := cmd.CombinedOutput()
	if len(output) == 0 {
		t.Fatal("persona produced no output")
	}
}

func TestBinaryUserHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "user")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	// user requires config, so it may fail, but should produce output
	output, _ := cmd.CombinedOutput()
	if len(output) == 0 {
		t.Fatal("user produced no output")
	}
}

func TestBinaryFullPath(t *testing.T) {
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cmd := exec.Command(absPath, "version")
	input, _ := os.Open("/dev/null")
	cmd.Stdin = input
	defer input.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version with absolute path failed: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("version produced no output")
	}
}
