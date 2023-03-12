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
	if prompt.Description != "Name a Git branch" {
		t.Errorf("expected description 'Name a Git branch', got %q", prompt.Description)
	}
	if len(prompt.Messages) != 2 {
		t.Errorf("expected 2 message, got %d", len(prompt.Messages))
	}
	if prompt.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %f", prompt.Temperature)
	}
	if prompt.isFoldEnabled() {
		t.Errorf("expected fold to be disabled")
	}
	if prompt.InputMarker != DefaultInputMarker {
		t.Errorf("expected input marker to be %q, got %q", DefaultInputMarker, prompt.InputMarker)
	}
	if prompt.OutputMarker != DefaultOutputMarker {
		t.Errorf("expected output marker to be %q, got %q", DefaultOutputMarker, prompt.OutputMarker)
	}
}

func TestLoadPromptsFold(t *testing.T) {
	prompt, err := NewPromptFromFile(filepath.Join("testdata", "fold.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt.Description != "Summarize" {
		t.Errorf("expected description 'Summarize', got %q", prompt.Description)
	}
	if len(prompt.Messages) != 2 {
		t.Errorf("expected 2 message, got %d", len(prompt.Messages))
	}
	if !prompt.isFoldEnabled() {
		t.Errorf("expected fold to be enabled")
	}
}

func TestSplitStringWithTokensLimit(t *testing.T) {
	str := "Hello, world!"
	tokens, err := splitStringWithTokensLimit(str, 2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0] != "Hello," {
		t.Errorf("expected 'Hello,', got %q", tokens[0])
	}
	if tokens[1] != " world!" {
		t.Errorf("expected 'world!', got %q", tokens[1])
	}
}
