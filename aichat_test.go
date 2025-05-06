package main

import (
	"testing"

	gogpt "github.com/sashabaranov/go-openai"
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
		{"o4-mini", 128 * 1024},
		{"o4-mini-2025-04-16", 128 * 1024},
		{"o3", 8 * 1024},
		{"o3-mini", 8 * 1024},
	}
	for _, d := range data {
		tokenLimit := tokenLimitOfModel(d.modelName)
		if tokenLimit != d.tokenLimit {
			t.Errorf("TokenLimitForModel(%q) returned %d, expected %d", d.modelName, tokenLimit, d.tokenLimit)
		}
	}
}

func TestHasBetaLimitations(t *testing.T) {
	tests := []struct {
		modelName   string
		hasLimits bool
	}{
		{"gpt-3.5-turbo", false},
		{"gpt-4", false},
		{"o4-mini", true},
		{"o4-mini-2025-04-16", true},
		{"o3", false},
		{"o3-mini", false},
	}
	
	for _, test := range tests {
		result := hasBetaLimitations(test.modelName)
		if result != test.hasLimits {
			t.Errorf("hasBetaLimitations(%q) returned %v, expected %v", test.modelName, result, test.hasLimits)
		}
	}
}

func TestApplyModelSpecificLimitations(t *testing.T) {
	request := gogpt.ChatCompletionRequest{
		Model:            "o4-mini",
		Temperature:      0.7,
		TopP:             0.9,
		N:                3,
		PresencePenalty:  0.5,
		FrequencyPenalty: 0.5,
	}
	
	applied := applyModelSpecificLimitations(&request, false)
	
	if !applied {
		t.Error("applyModelSpecificLimitations should return true for o4-mini model")
	}
	
	if request.Temperature != 1 {
		t.Errorf("Temperature should be 1, got %v", request.Temperature)
	}
	
	if request.TopP != 1 {
		t.Errorf("TopP should be 1, got %v", request.TopP)
	}
	
	if request.N != 1 {
		t.Errorf("N should be 1, got %v", request.N)
	}
	
	if request.PresencePenalty != 0 {
		t.Errorf("PresencePenalty should be 0, got %v", request.PresencePenalty)
	}
	
	if request.FrequencyPenalty != 0 {
		t.Errorf("FrequencyPenalty should be 0, got %v", request.FrequencyPenalty)
	}
	
	request = gogpt.ChatCompletionRequest{
		Model:            "gpt-4",
		Temperature:      0.7,
		TopP:             0.9,
		N:                3,
		PresencePenalty:  0.5,
		FrequencyPenalty: 0.5,
	}
	
	applied = applyModelSpecificLimitations(&request, false)
	
	if applied {
		t.Error("applyModelSpecificLimitations should return false for gpt-4 model")
	}
	
	if request.Temperature != 0.7 {
		t.Errorf("Temperature should remain 0.7, got %v", request.Temperature)
	}
}
