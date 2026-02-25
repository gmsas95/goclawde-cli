// Package editor provides utilities for opening files in the user's preferred editor.
package editor

import (
	"fmt"
	"os"
	"os/exec"
)

// DefaultEditor is the fallback editor if $EDITOR is not set.
const DefaultEditor = "nano"

// Open opens the specified file in the user's preferred editor.
// It respects the $EDITOR environment variable, falling back to nano.
// Returns an error if the editor cannot be found or the file cannot be opened.
func Open(filePath string) error {
	editor := GetEditor()

	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// OpenWithEditor opens the file with a specific editor command.
func OpenWithEditor(editor, filePath string) error {
	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GetEditor returns the user's preferred editor from $EDITOR environment variable,
// or the default editor if not set.
func GetEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return DefaultEditor
}

// OpenWithWait opens the file in the editor and waits for it to close.
// This is useful when you need to ensure the user has finished editing.
func OpenWithWait(filePath string) error {
	editor := GetEditor()

	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	return nil
}
