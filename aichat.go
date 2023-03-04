package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pborman/getopt/v2"
	gogpt "github.com/sashabaranov/go-gpt3"
)

type chatOptions struct {
	temperature float32
	maxTokens   int
}

type AIChat struct {
	client  *gogpt.Client
	options chatOptions
}

func (aiChat *AIChat) stdChatLoop() error {
	messages := []gogpt.ChatCompletionMessage{}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("user: ")
	for scanner.Scan() {
		messages = append(messages, gogpt.ChatCompletionMessage{
			Role:    gogpt.ChatMessageRoleUser,
			Content: scanner.Text(),
		})
		response, err := aiChat.client.CreateChatCompletion(context.Background(), gogpt.ChatCompletionRequest{
			Model:       gogpt.GPT3Dot5Turbo,
			Messages:    messages,
			Temperature: aiChat.options.temperature,
			MaxTokens:   aiChat.options.maxTokens,
		})
		if err != nil {
			return err
		}
		if len(response.Choices) == 0 {
			return fmt.Errorf("no choices")
		}
		fmt.Println("assistant: " + response.Choices[0].Message.Content)
		fmt.Print("user: ")
	}
	return scanner.Err()
}

func ListPrompts() error {
	prompts, err := ReadPrompts()
	if err != nil {
		return err
	}
	for name := range prompts {
		fmt.Println(name)
	}
	return nil
}

func main() {
	var temperature float32 = 0.5
	var maxTokens = 500
	var verbose = false
	var listPrompts = false
	getopt.Flag(&temperature, 't', "temperature", "temperature")
	getopt.Flag(&maxTokens, 'm', "max-tokens", "max tokens")
	getopt.Flag(&verbose, 'v', "verbose", "verbose")
	getopt.Flag(&listPrompts, 'l', "list-prompts", "list prompts")
	getopt.Parse()

	if listPrompts {
		if err := ListPrompts(); err != nil {
			log.Fatal(err)
		}
		return
	}

	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}
	options := chatOptions{
		temperature: temperature,
		maxTokens:   maxTokens,
	}
	if verbose {
		log.Printf("options: %+v", options)
	}
	aiChat := AIChat{
		client:  gogpt.NewClient(openaiAPIKey),
		options: options,
	}

	args := getopt.Args()
	if len(args) == 0 {
		if err := aiChat.stdChatLoop(); err != nil {
			log.Fatal(err)
		}
	} else {
		prompts, err := ReadPrompts()
		if err != nil {
			log.Fatal(err)
		}
		prompt := prompts[args[0]]
		if prompt == nil {
			log.Fatalf("prompt %q not found", args[0])
		}
		input := strings.Join(args[1:], " ")
		messages := prompt.CreateMessages(input)
		if verbose {
			log.Printf("messages: %+v", messages)
		}
		response, err := aiChat.client.CreateChatCompletion(context.Background(), gogpt.ChatCompletionRequest{
			Model:       gogpt.GPT3Dot5Turbo,
			Messages:    messages,
			Temperature: aiChat.options.temperature,
			MaxTokens:   aiChat.options.maxTokens,
		})
		if err != nil {
			log.Fatal(err)
		}
		if len(response.Choices) == 0 {
			log.Fatal("no choices")
		}
		fmt.Println(response.Choices[0].Message.Content)
	}

}
