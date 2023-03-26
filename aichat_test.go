package main

import (
	"testing"
)

func TestCountTokens(t *testing.T) {
	messages := []string{
		"Hello, world!",
		"How are you?",
	}
	count, err := CountTokens(messages)
	if err != nil {
		t.Errorf("CountTokens() returned an error: %v", err)
	}
	if count != 8 {
		t.Errorf("CountTokens() returned %d, expected 8", count)
	}
}

func TestTokenLimitOfModel(t *testing.T) {
	data := []struct {
		modelName  string
		tokenLimit int
	}{
		{"gpt-3.5-turbo", 4096},
		{"gpt-4", 8192},
	}
	for _, d := range data {
		tokenLimit := tokenLimitOfModel(d.modelName)
		if tokenLimit != d.tokenLimit {
			t.Errorf("TokenLimitForModel(%q) returned %d, expected %d", d.modelName, tokenLimit, d.tokenLimit)
		}
	}
}
