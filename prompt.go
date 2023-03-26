package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tokenizer "github.com/samber/go-gpt-3-encoder"
	gogpt "github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

const DefaultInputMarker = "$INPUT"
const DefaultOutputMarker = "$OUTPUT"

type Message struct {
	Role    string `yaml:"role"`
	Content string `yaml:"content"`
}

type Prompt struct {
	Description        string    `yaml:"description"`
	InputMarker        string    `yaml:"input_marker"`
	OutputMarker       string    `yaml:"output_marker"`
	Messages           []Message `yaml:"messages"`
	SubsequentMessages []Message `yaml:"subsequent_messages"`
	Temperature        float32   `yaml:"temperature"`
	MaxTokens          int       `yaml:"max_tokens"`
}

func (p *Prompt) isFoldEnabled() bool {
	return len(p.SubsequentMessages) > 0
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

func (p *Prompt) CreateSubsequentMessages(output, input string) []gogpt.ChatCompletionMessage {
	messages := []gogpt.ChatCompletionMessage{}
	for _, message := range p.SubsequentMessages {
		// replace input marker with input
		content := strings.ReplaceAll(message.Content, p.InputMarker, input)
		// replace output marker with output
		content = strings.ReplaceAll(content, p.OutputMarker, output)

		messages = append(messages, gogpt.ChatCompletionMessage{
			Role:    message.Role,
			Content: content,
		})
	}
	return messages
}

// CountTokens counts the number of tokens in the prompt
func (p *Prompt) CountTokens(encoder *tokenizer.Encoder) (int, error) {
	return countMessagesTokens(encoder, p.Messages)
}

func (p *Prompt) CountSubsequentTokens(encoder *tokenizer.Encoder) (int, error) {
	return countMessagesTokens(encoder, p.SubsequentMessages)
}

func countMessagesTokens(encoder *tokenizer.Encoder, messages []Message) (int, error) {
	count := 0
	for _, message := range messages {
		// Encode string with GPT tokenizer
		encoded, err := encoder.Encode(message.Content)
		if err != nil {
			return 0, err
		}
		count += len(encoded)
	}
	return count, nil
}

// AllowedInputTokens returns the number of tokens allowed for the input
func (p *Prompt) AllowedInputTokens(encoder *tokenizer.Encoder, tokenLimit, maxTokensOverride int, verbose bool) (int, error) {
	promptTokens, err := p.CountTokens(encoder)
	if err != nil {
		return 0, err
	}
	// reserve 500 tokens for output if maxTokens is not specified
	maxTokens := firstNonZeroInt(maxTokensOverride, p.MaxTokens, 500)
	result := tokenLimit - (promptTokens + maxTokens)
	if verbose {
		log.Printf("allowed tokens for input is %d", result)
	}
	if result <= 0 {
		return 0, fmt.Errorf("allowed tokens for input is %d, but it should be greater than 0", result)
	}
	return result, nil
}

func (p *Prompt) AllowedSubsequentInputTokens(encoder *tokenizer.Encoder, outputLen, tokenLimit, maxTokensOverride int, verbose bool) (int, error) {
	promptTokens, err := p.CountSubsequentTokens(encoder)
	if err != nil {
		return 0, err
	}
	// reserve 500 tokens for output if maxTokens is not specified
	maxTokens := firstNonZeroInt(maxTokensOverride, p.MaxTokens, 500)
	result := tokenLimit - (promptTokens + maxTokens + outputLen)
	if verbose {
		log.Printf("allowed tokens for subsequent input is %d", result)
	}
	if result <= 0 {
		return 0, fmt.Errorf("allowed tokens for subsequent input is %d, but it should be greater than 0", result)
	}
	return result, nil
}

func splitStringWithTokensLimit(s string, tokensLimit int) ([]string, error) {
	encoder, err := tokenizer.NewEncoder()
	if err != nil {
		return nil, err
	}
	encoded, err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}
	var parts []string
	for {
		if len(encoded) == 0 {
			break
		}
		if len(encoded) <= tokensLimit {
			parts = append(parts, encoder.Decode(encoded))
			break
		}
		parts = append(parts, encoder.Decode(encoded[:tokensLimit]))
		encoded = encoded[tokensLimit:]
	}
	return parts, nil
}

func (p *Prompt) CreateMessagesWithSplit(encoder *tokenizer.Encoder, input string, tokenLimit, maxTokensOverride int, verbose bool) ([][]gogpt.ChatCompletionMessage, error) {
	allowedInputTokens, err := p.AllowedInputTokens(encoder, tokenLimit, maxTokensOverride, verbose)
	if err != nil {
		return nil, err
	}
	inputParts, err := splitStringWithTokensLimit(input, allowedInputTokens)
	if err != nil {
		return nil, err
	}
	messages := [][]gogpt.ChatCompletionMessage{}
	for _, inputPart := range inputParts {
		messages = append(messages, p.CreateMessages(inputPart))
	}
	return messages, nil
}

func NewPromptFromFile(filename string) (*Prompt, error) {
	prompt := &Prompt{}
	if err := ReadYamlFromFile(filename, prompt); err != nil {
		return nil, err
	}
	if prompt.InputMarker == "" {
		prompt.InputMarker = DefaultInputMarker
	}
	if prompt.OutputMarker == "" {
		prompt.OutputMarker = DefaultOutputMarker
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
		// skip non-yaml files
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
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

func ListPrompts() error {
	prompts, err := ReadPrompts()
	if err != nil {
		return err
	}
	for name, prompt := range prompts {
		fmt.Printf("%s\t%s\n", name, prompt.Description)
	}
	return nil
}
