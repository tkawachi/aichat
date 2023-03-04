package main

import (
	"os"
	"path/filepath"
	"strings"

	gogpt "github.com/sashabaranov/go-gpt3"
	"gopkg.in/yaml.v3"
)

const DefaultInputMarker = "$INPUT"

type Prompt struct {
	InputMarker string `yaml:"input_marker"`
	Messages    []struct {
		Role    string `yaml:"role"`
		Content string `yaml:"content"`
	} `yaml:"messages"`
}

func (p *Prompt) CreateMessages(input string) []gogpt.ChatCompletionMessage {
	messages := []gogpt.ChatCompletionMessage{}
	for _, message := range p.Messages {
		// replace input marker with input
		content := strings.ReplaceAll(message.Content, p.InputMarker, input)

		messages = append(messages, gogpt.ChatCompletionMessage{
			Role:    message.Role,
			Content: content,
		})
	}
	return messages
}

func NewPromptFromFile(filename string) (*Prompt, error) {
	prompt := &Prompt{}
	if err := ReadYamlFromFile(filename, prompt); err != nil {
		return nil, err
	}
	if prompt.InputMarker == "" {
		prompt.InputMarker = DefaultInputMarker
	}
	return prompt, nil
}

func ReadYamlFromFile(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(v); err != nil {
		return err
	}
	return nil
}

func ReadPromptsInDir(dirname string) (map[string]*Prompt, error) {
	prompts := map[string]*Prompt{}
	files, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		prompt, err := NewPromptFromFile(dirname + "/" + file.Name())
		if err != nil {
			return nil, err
		}
		// strip extension
		name := strings.TrimSuffix(file.Name(), ".yaml")
		name = strings.TrimSuffix(name, ".yml")
		prompts[name] = prompt
	}
	return prompts, nil
}

func ReadPrompts() (map[string]*Prompt, error) {
	// Get HOME directory
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dirname := filepath.Join(home, ".aichat", "prompts")
	return ReadPromptsInDir(dirname)
}
