package persona

import (
	"testing"

	"go.uber.org/zap"
)

func TestProjectManagerCreateAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	logger := zap.NewNop()
	
	pm := NewProjectManager(tempDir, logger)
	
	// Create project
	project, err := pm.CreateProject("TestProject", "coding", "Test description")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	
	if project.Name != "TestProject" {
		t.Errorf("Expected name 'TestProject', got '%s'", project.Name)
	}
	if project.Type != "coding" {
		t.Errorf("Expected type 'coding', got '%s'", project.Type)
	}
	if project.Position != 1 {
		t.Errorf("Expected position 1, got %d", project.Position)
	}
	
	// Load project
	loaded, err := pm.LoadProject("TestProject")
	if err != nil {
		t.Fatalf("Failed to load project: %v", err)
	}
	
	if loaded.Name != "TestProject" {
		t.Error("Loaded project name mismatch")
	}
}

func TestProjectManagerListProjects(t *testing.T) {
	tempDir := t.TempDir()
	logger := zap.NewNop()
	
	pm := NewProjectManager(tempDir, logger)
	
	// Create multiple projects
	projects := []struct {
		name string
		typ  string
	}{
		{"Project1", "coding"},
		{"Project2", "writing"},
		{"Project3", "research"},
	}
	
	for _, p := range projects {
		_, err := pm.CreateProject(p.name, p.typ, "desc")
		if err != nil {
			t.Fatalf("Failed to create project %s: %v", p.name, err)
		}
	}
	
	// List projects
	list, err := pm.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}
	
	if len(list) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(list))
	}
	
	// Check LRU ordering (most recent first)
	if list[0].Name != "Project3" {
		t.Errorf("Expected Project3 first (LRU), got %s", list[0].Name)
	}
}

func TestProjectManagerArchive(t *testing.T) {
	tempDir := t.TempDir()
	logger := zap.NewNop()
	
	pm := NewProjectManager(tempDir, logger)
	
	// Create project
	_, err := pm.CreateProject("ToArchive", "coding", "desc")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	
	// Archive it
	err = pm.ArchiveProjectByName("ToArchive")
	if err != nil {
		t.Fatalf("Failed to archive project: %v", err)
	}
	
	// List should show archived
	list, _ := pm.ListProjects()
	found := false
	for _, p := range list {
		if p.Name == "ToArchive" && p.IsArchived {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Project not found in archived state")
	}
}

func TestProjectManagerDelete(t *testing.T) {
	tempDir := t.TempDir()
	logger := zap.NewNop()
	
	pm := NewProjectManager(tempDir, logger)
	
	// Create and delete
	_, err := pm.CreateProject("ToDelete", "coding", "desc")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	
	err = pm.DeleteProjectByName("ToDelete")
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}
	
	// Should not exist anymore
	_, err = pm.LoadProject("ToDelete")
	if err == nil {
		t.Error("Expected error loading deleted project")
	}
}

func TestSanitizeProjectName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Project", "my_project"},
		{"Test-123", "test-123"},
		{"Hello World!", "hello_world"},
		{"Special@#$Chars", "specialchars"},
	}
	
	for _, test := range tests {
		result := sanitizeProjectName(test.input)
		if result != test.expected {
			t.Errorf("sanitizeProjectName(%q) = %q, expected %q", 
				test.input, result, test.expected)
		}
	}
}

func TestGetProjectTemplate(t *testing.T) {
	// Coding template
	template := GetProjectTemplate("coding")
	if template == nil {
		t.Error("Coding template is nil")
	}
	if _, ok := template["language"]; !ok {
		t.Error("Coding template missing 'language' field")
	}
	
	// Unknown type
	unknown := GetProjectTemplate("unknown")
	if unknown == nil {
		t.Error("Unknown template should return default")
	}
}

func TestAvailableProjectTypes(t *testing.T) {
	types := AvailableProjectTypes()
	
	expected := []string{"coding", "writing", "research", "business"}
	if len(types) != len(expected) {
		t.Errorf("Expected %d types, got %d", len(expected), len(types))
	}
	
	// Check all expected types exist
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}
	
	for _, exp := range expected {
		if !typeMap[exp] {
			t.Errorf("Missing project type: %s", exp)
		}
	}
}
