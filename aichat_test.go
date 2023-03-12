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
