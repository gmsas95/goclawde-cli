package notes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/skills"
)

// NotesSkill provides note-taking functionality
type NotesSkill struct {
	*skills.BaseSkill
	notesDir string
}

// NewNotesSkill creates a new notes skill
func NewNotesSkill(notesDir string) *NotesSkill {
	if notesDir == "" {
		home, _ := os.UserHomeDir()
		notesDir = filepath.Join(home, ".jimmy", "notes")
	}

	// Create notes directory if it doesn't exist
	os.MkdirAll(notesDir, 0755)

	s := &NotesSkill{
		BaseSkill: skills.NewBaseSkill("notes", "Personal note-taking system", "1.0.0"),
		notesDir:  notesDir,
	}

	s.registerTools()
	return s
}

func (s *NotesSkill) registerTools() {
	s.AddTool(skills.Tool{
		Name:        "create_note",
		Description: "Create a new note",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Note title",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Note content (markdown supported)",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "Tags for the note",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"title", "content"},
		},
		Handler: s.handleCreateNote,
	})

	s.AddTool(skills.Tool{
		Name:        "read_note",
		Description: "Read a note by title",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Note title",
				},
			},
			"required": []string{"title"},
		},
		Handler: s.handleReadNote,
	})

	s.AddTool(skills.Tool{
		Name:        "list_notes",
		Description: "List all notes",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tag": map[string]interface{}{
					"type":        "string",
					"description": "Filter by tag",
				},
			},
		},
		Handler: s.handleListNotes,
	})

	s.AddTool(skills.Tool{
		Name:        "search_notes",
		Description: "Search notes by content",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
			},
			"required": []string{"query"},
		},
		Handler: s.handleSearchNotes,
	})

	s.AddTool(skills.Tool{
		Name:        "delete_note",
		Description: "Delete a note",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Note title",
				},
			},
			"required": []string{"title"},
		},
		Handler: s.handleDeleteNote,
	})
}

func (s *NotesSkill) sanitizeTitle(title string) string {
	// Remove special characters and spaces
	title = strings.ReplaceAll(title, " ", "_")
	title = strings.ReplaceAll(title, "/", "_")
	title = strings.ReplaceAll(title, "\\", "_")
	return strings.ToLower(title)
}

func (s *NotesSkill) handleCreateNote(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	tagsInterface, _ := args["tags"].([]interface{})

	if title == "" || content == "" {
		return nil, fmt.Errorf("title and content are required")
	}

	// Convert tags
	tags := make([]string, 0, len(tagsInterface))
	for _, t := range tagsInterface {
		if tag, ok := t.(string); ok {
			tags = append(tags, tag)
		}
	}

	// Build note content
	var noteContent strings.Builder
	noteContent.WriteString(fmt.Sprintf("# %s\n\n", title))
	noteContent.WriteString(fmt.Sprintf("Created: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	if len(tags) > 0 {
		noteContent.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(tags, ", ")))
	}
	noteContent.WriteString("\n---\n\n")
	noteContent.WriteString(content)

	// Save note
	filename := s.sanitizeTitle(title) + ".md"
	filepath := filepath.Join(s.notesDir, filename)

	if err := os.WriteFile(filepath, []byte(noteContent.String()), 0644); err != nil {
		return nil, fmt.Errorf("failed to save note: %w", err)
	}

	return map[string]interface{}{
		"title":    title,
		"filename": filename,
		"saved_at": filepath,
	}, nil
}

func (s *NotesSkill) handleReadNote(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	title, _ := args["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	filename := s.sanitizeTitle(title) + ".md"
	filepath := filepath.Join(s.notesDir, filename)

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("note not found: %s", title)
	}

	return map[string]interface{}{
		"title":   title,
		"content": string(content),
	}, nil
}

func (s *NotesSkill) handleListNotes(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	tag, _ := args["tag"].(string)

	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	notes := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			info, _ := entry.Info()
			name := strings.TrimSuffix(entry.Name(), ".md")

			// Read file to check tags if filtering
			if tag != "" {
				content, err := os.ReadFile(filepath.Join(s.notesDir, entry.Name()))
				if err == nil && !strings.Contains(string(content), "Tags: "+tag) &&
					!strings.Contains(string(content), "Tags: "+strings.Join([]string{"", tag}, ", ")) {
					continue
				}
			}

			notes = append(notes, map[string]interface{}{
				"title":      name,
				"filename":   entry.Name(),
				"updated_at": info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	return notes, nil
}

func (s *NotesSkill) handleSearchNotes(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	query = strings.ToLower(query)
	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to search notes: %w", err)
	}

	results := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			content, err := os.ReadFile(filepath.Join(s.notesDir, entry.Name()))
			if err != nil {
				continue
			}

			if strings.Contains(strings.ToLower(string(content)), query) {
				name := strings.TrimSuffix(entry.Name(), ".md")
				info, _ := entry.Info()
				results = append(results, map[string]interface{}{
					"title":      name,
					"filename":   entry.Name(),
					"updated_at": info.ModTime().Format("2006-01-02 15:04:05"),
				})
			}
		}
	}

	return results, nil
}

func (s *NotesSkill) handleDeleteNote(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	title, _ := args["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	filename := s.sanitizeTitle(title) + ".md"
	filepath := filepath.Join(s.notesDir, filename)

	if err := os.Remove(filepath); err != nil {
		return nil, fmt.Errorf("failed to delete note: %w", err)
	}

	return fmt.Sprintf("Note '%s' deleted successfully", title), nil
}
