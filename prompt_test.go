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
