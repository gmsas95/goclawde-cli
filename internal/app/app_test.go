package app

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "create app with version",
			version: "1.0.0",
		},
		{
			name:    "create app with dev version",
			version: "dev",
		},
		{
			name:    "create app with empty version",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(nil, nil, nil, nil, tt.version)
			if app == nil {
				t.Fatal("expected app to be created, got nil")
			}
			if app.Version != tt.version {
				t.Errorf("expected version %q, got %q", tt.version, app.Version)
			}
		})
	}
}

func TestSetSkillsRegistry(t *testing.T) {
	app := New(nil, nil, nil, nil, "test")

	app.SetSkillsRegistry(nil)
	if app.SkillsRegistry != nil {
		t.Error("expected skills registry to be nil")
	}
}

func TestPrintInteractiveHelp(t *testing.T) {
	PrintInteractiveHelp()
}
