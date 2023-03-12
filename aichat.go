package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/pborman/getopt/v2"
	tokenizer "github.com/samber/go-gpt-3-encoder"
	gogpt "github.com/sashabaranov/go-gpt3"
)

type chatOptions struct {
	temperature  float32
	maxTokens    int
	nonStreaming bool
}

type AIChat struct {
	client  *gogpt.Client
	options chatOptions
}

// stramCompletion print out the chat completion in streaming mode.
func streamCompletion(client *gogpt.Client, request gogpt.ChatCompletionRequest, out io.Writer) error {
	stream, err := client.CreateChatCompletionStream(context.Background(), request)
	if err != nil {
		return err
	}
	defer stream.Close()
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println()
			break
		}
		if err != nil {
			return fmt.Errorf("stream recv: %w", err)
		}
		if len(response.Choices) == 0 {
			return fmt.Errorf("no choices returned")
		}
		_, err = fmt.Fprint(out, response.Choices[0].Delta.Content)
		if err != nil {
			return err
		}
	}
	return nil
}

// stramCompletion print out the chat completion in non-streaming mode.
func nonStreamCompletion(client *gogpt.Client, request gogpt.ChatCompletionRequest, out io.Writer) error {
	response, err := client.CreateChatCompletion(context.Background(), request)
	if err != nil {
		return err
	}
	if len(response.Choices) == 0 {
		return fmt.Errorf("no choices returned")
	}
	_, err = fmt.Fprint(out, response.Choices[0].Message.Content+"\n")
	return err
}

func (aiChat *AIChat) stdChatLoop() error {
	messages := []gogpt.ChatCompletionMessage{}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("user: ")
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Println("Empty input. Exiting...")
			return nil
		}
		messages = append(messages, gogpt.ChatCompletionMessage{
			Role:    gogpt.ChatMessageRoleUser,
			Content: input,
		})
		fmt.Print("assistant: ")
		request := gogpt.ChatCompletionRequest{
			Model:       gogpt.GPT3Dot5Turbo,
			Messages:    messages,
			Temperature: aiChat.options.temperature,
			MaxTokens:   aiChat.options.maxTokens,
		}
		var err error
		if aiChat.options.nonStreaming {
			err = nonStreamCompletion(aiChat.client, request, os.Stdout)
		} else {
			err = streamCompletion(aiChat.client, request, os.Stdout)
		}
		if err != nil {
			return err
		}
		fmt.Print("user: ")
	}
	return scanner.Err()
}

func firstNonZeroInt(i ...int) int {
	for _, v := range i {
		if v != 0 {
			return v
		}
	}
	return 0
}

func firstNonZeroFloat32(f ...float32) float32 {
	for _, v := range f {
		if v != 0 {
			return v
		}
	}
	return 0
}

func main() {
	var temperature float32 = 0.5
	var maxTokens = 500
	var verbose = false
	var listPrompts = false
	var nonStreaming = false
	var split = false
	getopt.FlagLong(&temperature, "temperature", 't', "temperature")
	getopt.FlagLong(&maxTokens, "max-tokens", 'm', "max tokens")
	getopt.FlagLong(&verbose, "verbose", 'v', "verbose output")
	getopt.FlagLong(&listPrompts, "list-prompts", 'l', "list prompts")
	getopt.FlagLong(&nonStreaming, "non-streaming", 0, "non streaming mode")
	getopt.FlagLong(&split, "split", 0, "split input")
	getopt.Parse()

	if listPrompts {
		if err := ListPrompts(); err != nil {
			log.Fatal(err)
		}
		return
	}

	openaiAPIKey, err := ReadOpenAIAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	options := chatOptions{
		temperature:  temperature,
		maxTokens:    maxTokens,
		nonStreaming: nonStreaming,
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
		// read all from Stdin
		input := scanAll(bufio.NewScanner(os.Stdin))

		var messagesSlice [][]gogpt.ChatCompletionMessage

		if split {
			messagesSlice, err = prompt.CreateMessagesWithSplit(input, 0) // TODO pass maxTokens if it is specified via command line flag
			if err != nil {
				log.Fatal(err)
			}
			if verbose {
				log.Printf("messages was split to %d parts", len(messagesSlice))

			}
		} else {
			messages := prompt.CreateMessages(input)
			if verbose {
				log.Printf("messages: %+v", messagesSlice)
			}
			messagesSlice = [][]gogpt.ChatCompletionMessage{messages}
		}

		for _, messages := range messagesSlice {

			maxTokens := firstNonZeroInt(prompt.MaxTokens, aiChat.options.maxTokens)

			request := gogpt.ChatCompletionRequest{
				Model:       gogpt.GPT3Dot5Turbo,
				Messages:    messages,
				Temperature: firstNonZeroFloat32(prompt.Temperature, aiChat.options.temperature),
				MaxTokens:   maxTokens,
			}

			cnt, err := CountTokens(mapSlice(messages, func(m gogpt.ChatCompletionMessage) string { return m.Content }))
			if err != nil {
				log.Fatal(err)
			}
			if cnt > 4096 {
				log.Fatalf("total tokens %d exceeds 4096", cnt)
			}

			if aiChat.options.nonStreaming {
				err = nonStreamCompletion(aiChat.client, request, os.Stdout)
			} else {
				err = streamCompletion(aiChat.client, request, os.Stdout)
			}
			if err != nil {
				log.Fatal(err)
			}
		}
	}

}

func scanAll(scanner *bufio.Scanner) string {
	input := ""
	for scanner.Scan() {
		input += scanner.Text() + "\n"
	}
	return input
}

// mapSlice maps a slice of type T to a slice of type M using the function f.
func mapSlice[T any, M any](a []T, f func(T) M) []M {
	r := make([]M, len(a))
	for i, v := range a {
		r[i] = f(v)
	}
	return r
}

// CountTokens returns the number of tokens in the messages.
func CountTokens(messages []string) (int, error) {
	count := 0
	encoder, err := tokenizer.NewEncoder()
	if err != nil {
		return 0, err
	}
	for _, message := range messages {
		// Encode string with GPT tokenizer
		encoded, err := encoder.Encode(message)
		if err != nil {
			return 0, err
		}
		count += len(encoded)
	}
	return count, nil
}
