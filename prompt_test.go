package main

import (
	"path/filepath"
	"testing"
)

func TestLoadPrompts(t *testing.T) {
	prompt, err := NewPromptFromFile(filepath.Join("testdata", "name-branch.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt.Description == "" {
		t.Errorf("expected description, got empty string")
	}
}
