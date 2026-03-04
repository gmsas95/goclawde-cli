package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors SKILL.md files for changes and triggers hot-reload
type Watcher struct {
	watcher     *fsnotify.Watcher
	loader      *SkillLoader
	watchedDirs map[string]bool
	mu          sync.RWMutex
	onReload    func(skillName string)
	onError     func(skillName string, err error)
	debounce    time.Duration
}

// SkillLoader handles loading and reloading of skills
type SkillLoader struct {
	registry  *EnhancedRegistry
	skillsDir string
	mu        sync.RWMutex
}

// NewSkillLoader creates a new skill loader
func NewSkillLoader(registry *EnhancedRegistry, skillsDir string) *SkillLoader {
	return &SkillLoader{
		registry:  registry,
		skillsDir: skillsDir,
	}
}

// LoadSkill loads a skill from a SKILL.md file
func (sl *SkillLoader) LoadSkill(skillPath string) (*RuntimeSkill, error) {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	manifest, docs, err := ParseSkillMarkdown(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}

	_ = docs // Documentation can be used for help text

	// Determine source from path
	source := SourceLocal
	if strings.Contains(skillPath, "github.com") {
		source = SourceGitHub
	}

	skill := &RuntimeSkill{
		Manifest:  manifest,
		Status:    StatusEnabled,
		Source:    source,
		SourceURL: skillPath,
		LoadedAt:  time.Now(),
		UpdatedAt: time.Now(),
		Tools:     make([]Tool, 0, len(manifest.Tools)),
	}

	// Convert manifest tools to runtime tools
	for _, manifestTool := range manifest.Tools {
		tool := Tool{
			Name:        manifestTool.Name,
			Description: manifestTool.Description,
			Parameters:  manifestTool.ToJSONSchema(),
			Handler:     sl.createToolHandler(manifest.Name, manifestTool),
		}
		skill.Tools = append(skill.Tools, tool)
	}

	return skill, nil
}

// ReloadSkill reloads a skill from a file path
func (sl *SkillLoader) ReloadSkill(skillPath string) error {
	skill, err := sl.LoadSkill(skillPath)
	if err != nil {
		return err
	}

	// Check if skill already exists
	if existing, ok := sl.registry.GetRuntimeSkill(skill.Manifest.Name); ok {
		// Preserve usage stats
		skill.UseCount = existing.UseCount
		skill.LastUsedAt = existing.LastUsedAt

		// Update the skill
		if err := sl.registry.UpdateSkill(existing.Manifest.Name, skill); err != nil {
			return fmt.Errorf("failed to update skill: %w", err)
		}
	} else {
		// Register new skill
		if err := sl.registry.RegisterSkill(skill); err != nil {
			return fmt.Errorf("failed to register skill: %w", err)
		}
	}

	log.Printf("[Hot-Reload] Skill '%s' reloaded successfully", skill.Manifest.Name)
	return nil
}

// UnloadSkill removes a skill by file path
func (sl *SkillLoader) UnloadSkill(skillPath string) error {
	// Find skill by path
	sl.registry.mu.RLock()
	var skillName string
	for name, skill := range sl.registry.runtimeSkills {
		if skill.SourceURL == skillPath {
			skillName = name
			break
		}
	}
	sl.registry.mu.RUnlock()

	if skillName == "" {
		return fmt.Errorf("skill not found for path: %s", skillPath)
	}

	if err := sl.registry.UnregisterSkill(skillName); err != nil {
		return fmt.Errorf("failed to unload skill: %w", err)
	}

	log.Printf("[Hot-Reload] Skill '%s' unloaded", skillName)
	return nil
}

// LoadAllSkills loads all skills from the skills directory
func (sl *SkillLoader) LoadAllSkills() error {
	entries, err := os.ReadDir(sl.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet
		}
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillPath := filepath.Join(sl.skillsDir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				if _, err := sl.LoadSkill(skillPath); err != nil {
					log.Printf("[Loader] Failed to load skill %s: %v", entry.Name(), err)
				} else {
					log.Printf("[Loader] Loaded skill: %s", entry.Name())
				}
			}
		}
	}

	return nil
}

// createToolHandler creates a handler for a manifest tool
func (sl *SkillLoader) createToolHandler(skillName string, tool ManifestTool) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Validate required arguments
		for _, param := range tool.Parameters {
			if param.Required {
				if _, ok := args[param.Name]; !ok {
					return nil, fmt.Errorf("missing required parameter: %s", param.Name)
				}
			}
		}

		// Get the skill directory
		skillDir := filepath.Join(sl.skillsDir, skillName)

		// Check if skill has an entry point script
		entryPoint := filepath.Join(skillDir, "skill.py")
		if _, err := os.Stat(entryPoint); os.IsNotExist(err) {
			// Try other entry points
			entryPoint = filepath.Join(skillDir, "skill.js")
			if _, err := os.Stat(entryPoint); os.IsNotExist(err) {
				return nil, fmt.Errorf("no entry point found for skill %s", skillName)
			}
		}

		// Execute the tool
		switch filepath.Ext(entryPoint) {
		case ".py":
			return sl.executePythonTool(ctx, entryPoint, tool.Name, args)
		case ".js":
			return sl.executeNodeTool(ctx, entryPoint, tool.Name, args)
		default:
			return nil, fmt.Errorf("unsupported skill type: %s", filepath.Ext(entryPoint))
		}
	}
}

// executePythonTool executes a Python-based tool
func (sl *SkillLoader) executePythonTool(ctx context.Context, entryPoint, toolName string, args map[string]interface{}) (interface{}, error) {
	argsJSON, _ := json.Marshal(args)

	cmd := exec.CommandContext(ctx, "python3", entryPoint, toolName, string(argsJSON))
	cmd.Dir = filepath.Dir(entryPoint)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w (output: %s)", err, string(output))
	}

	// Try to parse as JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		// Return as text if not JSON
		return map[string]interface{}{
			"output": string(output),
		}, nil
	}

	return result, nil
}

// executeNodeTool executes a Node.js-based tool
func (sl *SkillLoader) executeNodeTool(ctx context.Context, entryPoint, toolName string, args map[string]interface{}) (interface{}, error) {
	argsJSON, _ := json.Marshal(args)

	cmd := exec.CommandContext(ctx, "node", entryPoint, toolName, string(argsJSON))
	cmd.Dir = filepath.Dir(entryPoint)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w (output: %s)", err, string(output))
	}

	// Try to parse as JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		// Return as text if not JSON
		return map[string]interface{}{
			"output": string(output),
		}, nil
	}

	return result, nil
}

// NewWatcher creates a new file watcher for hot-reload
func NewWatcher(loader *SkillLoader) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &Watcher{
		watcher:     fsWatcher,
		loader:      loader,
		watchedDirs: make(map[string]bool),
		debounce:    500 * time.Millisecond,
	}, nil
}

// SetDebounce sets the debounce duration
func (w *Watcher) SetDebounce(d time.Duration) {
	w.debounce = d
}

// SetCallbacks sets the reload and error callbacks
func (w *Watcher) SetCallbacks(onReload func(string), onError func(string, error)) {
	onReload = onReload
	onError = onError
}

// WatchDirectory starts watching a directory for SKILL.md changes
func (w *Watcher) WatchDirectory(dir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.watchedDirs[dir] {
		return nil // Already watching
	}

	// Walk the directory tree
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch %s: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	w.watchedDirs[dir] = true
	log.Printf("[Watcher] Watching directory: %s", dir)

	// Start the event loop
	go w.eventLoop()

	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	return w.watcher.Close()
}

// eventLoop processes file system events
func (w *Watcher) eventLoop() {
	debounceTimers := make(map[string]*time.Timer)

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only process SKILL.md files
			if !strings.HasSuffix(event.Name, "SKILL.md") {
				continue
			}

			// Debounce events
			if timer, exists := debounceTimers[event.Name]; exists {
				timer.Stop()
			}

			debounceTimers[event.Name] = time.AfterFunc(w.debounce, func() {
				w.handleEvent(event)
				delete(debounceTimers, event.Name)
			})

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[Watcher] Error: %v", err)
		}
	}
}

// handleEvent processes a single file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		log.Printf("[Watcher] File modified: %s", event.Name)
		if err := w.loader.ReloadSkill(event.Name); err != nil {
			log.Printf("[Watcher] Failed to reload skill: %v", err)
			if w.onError != nil {
				w.onError(event.Name, err)
			}
		} else {
			if w.onReload != nil {
				w.onReload(event.Name)
			}
		}

	case event.Op&fsnotify.Create == fsnotify.Create:
		log.Printf("[Watcher] File created: %s", event.Name)
		if err := w.loader.ReloadSkill(event.Name); err != nil {
			log.Printf("[Watcher] Failed to load new skill: %v", err)
			if w.onError != nil {
				w.onError(event.Name, err)
			}
		} else {
			if w.onReload != nil {
				w.onReload(event.Name)
			}
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		log.Printf("[Watcher] File removed: %s", event.Name)
		if err := w.loader.UnloadSkill(event.Name); err != nil {
			log.Printf("[Watcher] Failed to unload skill: %v", err)
			if w.onError != nil {
				w.onError(event.Name, err)
			}
		}
	}
}

// IsWatching returns true if the directory is being watched
func (w *Watcher) IsWatching(dir string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watchedDirs[dir]
}

// GetWatchedDirs returns all watched directories
func (w *Watcher) GetWatchedDirs() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	dirs := make([]string, 0, len(w.watchedDirs))
	for dir := range w.watchedDirs {
		dirs = append(dirs, dir)
	}
	return dirs
}
