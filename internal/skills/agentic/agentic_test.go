package agentic

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAgenticSkill(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	if skill == nil {
		t.Fatal("expected skill to be created")
	}
	if skill.Name() != "agentic" {
		t.Errorf("expected name 'agentic', got %q", skill.Name())
	}
	if skill.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", skill.Version())
	}
}

func TestToolsRegistered(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	tools := skill.Tools()

	expectedTools := []string{
		"get_system_resources",
		"list_processes",
		"get_network_info",
		"get_environment",
		"analyze_project_structure",
		"analyze_code_file",
		"search_code",
		"find_todos",
		"git_status",
		"git_log",
		"git_diff",
		"git_blame",
		"create_task_plan",
		"reflect_on_task",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("expected tool %q to be registered", expected)
		}
	}
}

func TestHandleGetEnvironment(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	result, err := skill.handleGetEnvironment(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env, ok := result.(map[string]string)
	if !ok {
		t.Fatal("expected map[string]string result")
	}

	if _, exists := env["GO_VERSION"]; !exists {
		t.Error("expected GO_VERSION in result")
	}
}

func TestHandleGetEnvironmentWithFilter(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	result, err := skill.handleGetEnvironment(ctx, map[string]interface{}{
		"filter": "GO",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env, ok := result.(map[string]string)
	if !ok {
		t.Fatal("expected map[string]string result")
	}

	for key := range env {
		if len(key) < 2 || key[:2] != "GO" {
			t.Errorf("expected key to start with 'GO', got %q", key)
		}
	}
}

func TestHandleCreateTaskPlan(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	result, err := skill.handleCreateTaskPlan(ctx, map[string]interface{}{
		"goal":  "Test goal",
		"steps": []interface{}{"step1", "step2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map[string]interface{} result")
	}

	if plan["goal"] != "Test goal" {
		t.Errorf("expected goal 'Test goal', got %v", plan["goal"])
	}
	if plan["status"] != "planning" {
		t.Errorf("expected status 'planning', got %v", plan["status"])
	}
}

func TestHandleCreateTaskPlanMissingGoal(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	_, err := skill.handleCreateTaskPlan(ctx, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing goal")
	}
}

func TestHandleReflectOnTask(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	result, err := skill.handleReflectOnTask(ctx, map[string]interface{}{
		"task":          "Test task",
		"current_state": "in progress",
		"actions_taken": []interface{}{"action1", "action2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reflection, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map[string]interface{} result")
	}

	if reflection["task"] != "Test task" {
		t.Errorf("expected task 'Test task', got %v", reflection["task"])
	}
	suggestions, ok := reflection["suggestions"].([]string)
	if !ok || len(suggestions) == 0 {
		t.Error("expected suggestions to be non-empty")
	}
}

func TestHandleAnalyzeCodeFileMissingPath(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	_, err := skill.handleAnalyzeCodeFile(ctx, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestHandleAnalyzeCodeFileNonexistent(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	_, err := skill.handleAnalyzeCodeFile(ctx, map[string]interface{}{
		"path": "/nonexistent/file.go",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestHandleAnalyzeCodeFileGoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}

type MyStruct struct {
	Name string
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	result, err := skill.handleAnalyzeCodeFile(ctx, map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	analysis, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map[string]interface{} result")
	}

	if analysis["package"] != "main" {
		t.Errorf("expected package 'main', got %v", analysis["package"])
	}

	imports, ok := analysis["imports"].([]string)
	if !ok || len(imports) == 0 {
		t.Error("expected imports to contain 'fmt'")
	}
}

func TestHandleSearchCodeMissingPattern(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	_, err := skill.handleSearchCode(ctx, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing pattern")
	}
}

func TestHandleGitStatusNotRepo(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	_, err := skill.handleGitStatus(ctx, map[string]interface{}{
		"path": "/tmp",
	})
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestHandleGitBlameMissingFile(t *testing.T) {
	skill := NewAgenticSkill("/tmp")
	ctx := context.Background()

	_, err := skill.handleGitBlame(ctx, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExtractTODOs(t *testing.T) {
	content := `package main
// TODO: implement this
func main() {
	// FIXME: broken
	// NOTE: important
}
`
	todos := extractTODOs(content)

	if len(todos) < 2 {
		t.Errorf("expected at least 2 todos, got %d", len(todos))
	}

	foundTODO := false
	foundFIXME := false
	for _, todo := range todos {
		if todo["type"] == "TODO" {
			foundTODO = true
		}
		if todo["type"] == "FIXME" {
			foundFIXME = true
		}
	}

	if !foundTODO {
		t.Error("expected to find TODO")
	}
	if !foundFIXME {
		t.Error("expected to find FIXME")
	}
}

func TestDetectLanguages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.py"), []byte("print('hi')"), 0644)

	langs := detectLanguages(tmpDir)

	foundGo := false
	foundPython := false
	for _, lang := range langs {
		if lang == "Go" {
			foundGo = true
		}
		if lang == "Python" {
			foundPython = true
		}
	}

	if !foundGo {
		t.Error("expected to detect Go")
	}
	if !foundPython {
		t.Error("expected to detect Python")
	}
}

func TestDetectFrameworks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0644)

	frameworks := detectFrameworks(tmpDir)

	foundGoMods := false
	foundDocker := false
	for _, fw := range frameworks {
		if fw == "Go Modules" {
			foundGoMods = true
		}
		if fw == "Docker" {
			foundDocker = true
		}
	}

	if !foundGoMods {
		t.Error("expected to detect Go Modules")
	}
	if !foundDocker {
		t.Error("expected to detect Docker")
	}
}

func TestCountFileTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "c.py"), []byte("print('hi')"), 0644)

	counts := countFileTypes(tmpDir)

	if counts[".go"] != 2 {
		t.Errorf("expected 2 .go files, got %d", counts[".go"])
	}
	if counts[".py"] != 1 {
		t.Errorf("expected 1 .py file, got %d", counts[".py"])
	}
}
