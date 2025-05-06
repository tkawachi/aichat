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
	"time"

	"github.com/pborman/getopt/v2"
	tokenizer "github.com/samber/go-gpt-3-encoder"
	gogpt "github.com/sashabaranov/go-openai"
)

type chatOptions struct {
	model        string
	temperature  float32
	maxTokens    int
	nonStreaming bool
	verbose      bool
	saveHistory  bool
	loadHistory  string
	listHistory  bool
	deleteHistory string
}

type AIChat struct {
	client       *gogpt.Client
	encoder      *tokenizer.Encoder
	options      chatOptions
	conversation *Conversation
}

// streamCompletion print out the chat completion in streaming mode.
func streamCompletion(client *gogpt.Client, request gogpt.ChatCompletionRequest, out io.Writer, verbose bool) error {
	applyModelSpecificLimitations(&request, verbose)
	
	stream, err := client.CreateChatCompletionStream(context.Background(), request)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			break
		}
		if err != nil {
			return fmt.Errorf("stream recv: %w", err)
		}
		if len(response.Choices) == 0 {
			if verbose {
				log.Println("no choices returned")
			}
			continue
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
	applyModelSpecificLimitations(&request, false)
	
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
	var messages []gogpt.ChatCompletionMessage
	
	if aiChat.conversation == nil {
		aiChat.conversation = NewConversation("New Conversation", aiChat.options.model)
	} else {
		// Use messages from loaded conversation
		messages = aiChat.conversation.ToGPTMessages()
	}
	
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("user: ")
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Println("Empty input. Exiting...")
			
			if aiChat.options.saveHistory && len(aiChat.conversation.Messages) > 0 {
				if err := SaveConversation(aiChat.conversation); err != nil {
					log.Printf("Failed to save conversation: %v", err)
				} else if aiChat.options.verbose {
					log.Printf("Conversation saved with ID: %s", aiChat.conversation.ID)
				}
			}
			
			return nil
		}
		
		if strings.HasPrefix(input, "/") {
			cmd := strings.TrimSpace(strings.TrimPrefix(input, "/"))
			switch {
			case cmd == "save":
				if len(aiChat.conversation.Messages) == 0 {
					fmt.Println("No messages to save.")
				} else {
					if err := SaveConversation(aiChat.conversation); err != nil {
						fmt.Printf("Error saving conversation: %v\n", err)
					} else {
						fmt.Printf("Conversation saved with ID: %s\n", aiChat.conversation.ID)
					}
				}
				fmt.Print("user: ")
				continue
				
			case cmd == "list":
				conversations, err := ListConversations()
				if err != nil {
					fmt.Printf("Error listing conversations: %v\n", err)
				} else if len(conversations) == 0 {
					fmt.Println("No saved conversations.")
				} else {
					fmt.Println("Saved conversations:")
					for _, conv := range conversations {
						fmt.Printf("- %s: %s (%s)\n", conv.ID, conv.Title, conv.UpdatedAt.Format(time.RFC3339))
					}
				}
				fmt.Print("user: ")
				continue
				
			case strings.HasPrefix(cmd, "load "):
				id := strings.TrimSpace(strings.TrimPrefix(cmd, "load "))
				conv, err := LoadConversation(id)
				if err != nil {
					fmt.Printf("Error loading conversation: %v\n", err)
				} else {
					aiChat.conversation = conv
					messages = conv.ToGPTMessages()
					fmt.Printf("Loaded conversation: %s\n", conv.Title)
				}
				fmt.Print("user: ")
				continue
				
			case strings.HasPrefix(cmd, "delete "):
				id := strings.TrimSpace(strings.TrimPrefix(cmd, "delete "))
				if err := DeleteConversation(id); err != nil {
					fmt.Printf("Error deleting conversation: %v\n", err)
				} else {
					fmt.Println("Conversation deleted.")
				}
				fmt.Print("user: ")
				continue
				
			case cmd == "help":
				fmt.Println("Available commands:")
				fmt.Println("  /save              - Save the current conversation")
				fmt.Println("  /list              - List all saved conversations")
				fmt.Println("  /load <id>         - Load a conversation by ID")
				fmt.Println("  /delete <id>       - Delete a conversation by ID")
				fmt.Println("  /help              - Show this help message")
				fmt.Print("user: ")
				continue
			}
		}
		
		// Add user message to conversation
		aiChat.conversation.AddMessage(gogpt.ChatMessageRoleUser, input)
		
		messages = append(messages, gogpt.ChatCompletionMessage{
			Role:    gogpt.ChatMessageRoleUser,
			Content: input,
		})
		
		fmt.Print("assistant: ")
		request := gogpt.ChatCompletionRequest{
			Model:       aiChat.options.model,
			Messages:    messages,
			Temperature: aiChat.options.temperature,
			MaxTokens:   aiChat.options.maxTokens,
		}
		
		var assistantResponse string
		var err error
		
	if aiChat.options.nonStreaming {
		applyModelSpecificLimitations(&request, aiChat.options.verbose)
		
		response, err := aiChat.client.CreateChatCompletion(context.Background(), request)
		if err != nil {
			return err
		}
			if len(response.Choices) == 0 {
				return fmt.Errorf("no choices returned")
			}
			assistantResponse = response.Choices[0].Message.Content
			fmt.Println(assistantResponse)
		} else {
			var responseBuilder strings.Builder
			
			writer := io.MultiWriter(os.Stdout, &responseBuilder)
			
			err = streamCompletion(aiChat.client, request, writer, aiChat.options.verbose)
			if err != nil {
				return err
			}
			
			assistantResponse = responseBuilder.String()
		}
		
		// Add assistant response to conversation
		aiChat.conversation.AddMessage(gogpt.ChatMessageRoleAssistant, assistantResponse)
		
		messages = append(messages, gogpt.ChatCompletionMessage{
			Role:    gogpt.ChatMessageRoleAssistant,
			Content: assistantResponse,
		})
		
		if aiChat.conversation.Title == "New Conversation" && len(aiChat.conversation.Messages) == 2 {
			aiChat.conversation.Title = GetConversationTitle(messages)
		}
		
		fmt.Print("user: ")
	}
	
	if aiChat.options.saveHistory && len(aiChat.conversation.Messages) > 0 {
		if err := SaveConversation(aiChat.conversation); err != nil {
			log.Printf("Failed to save conversation: %v", err)
		} else if aiChat.options.verbose {
			log.Printf("Conversation saved with ID: %s", aiChat.conversation.ID)
		}
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

func (aiChat *AIChat) fold(prompt *Prompt, input string) error {
	encoded, err := aiChat.encoder.Encode(input)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	tokenLimit := tokenLimitOfModel(aiChat.options.model)
	firstAllowedTokens, err := prompt.AllowedInputTokens(aiChat.encoder, tokenLimit, aiChat.options.maxTokens, aiChat.options.verbose)
	if err != nil {
		return err
	}
	idx := firstAllowedTokens
	for idx > len(encoded) {
		idx = len(encoded)
	}

	firstEncoded := encoded[:idx]
	firstInput := aiChat.encoder.Decode(firstEncoded)
	temperature := firstNonZeroFloat32(aiChat.options.temperature, prompt.Temperature)
	firstRequest := gogpt.ChatCompletionRequest{
		Model:       aiChat.options.model,
		Messages:    prompt.CreateMessages(firstInput),
		Temperature: temperature,
	}
	if aiChat.options.verbose {
		log.Printf("first request: %+v", firstRequest)
	}
	
	applyModelSpecificLimitations(&firstRequest, aiChat.options.verbose)
	
	response, err := aiChat.client.CreateChatCompletion(context.Background(), firstRequest)
	if err != nil {
		return fmt.Errorf("create chat completion: %w", err)
	}
	if len(response.Choices) == 0 {
		return fmt.Errorf("no choices returned")
	}
	output := response.Choices[0].Message.Content
	if idx >= len(encoded) {
		fmt.Println(output)
		return nil
	}
	if aiChat.options.verbose {
		log.Printf("first output: %s", output)
	}

	for idx < len(encoded) {
		outputTokens, err := aiChat.encoder.Encode(output)
		if err != nil {
			return fmt.Errorf("encode: %w", err)
		}

		allowedTokens, err := prompt.AllowedSubsequentInputTokens(
			aiChat.encoder, len(outputTokens), tokenLimit, aiChat.options.maxTokens, aiChat.options.verbose)
		if err != nil {
			return fmt.Errorf("allowed subsequent input tokens: %w", err)
		}
		nextIdx := idx + allowedTokens
		if nextIdx > len(encoded) {
			nextIdx = len(encoded)
		}
		input := aiChat.encoder.Decode(encoded[idx:nextIdx])
		request := gogpt.ChatCompletionRequest{
			Model:       aiChat.options.model,
			Messages:    prompt.CreateSubsequentMessages(output, input),
			Temperature: temperature,
		}
	if aiChat.options.verbose {
		log.Printf("subsequent request: %+v", request)
	}
	
	applyModelSpecificLimitations(&request, aiChat.options.verbose)
	
	response, err := aiChat.client.CreateChatCompletion(context.Background(), request)
	if err != nil {
			return fmt.Errorf("create chat completion: %w", err)
		}
		if len(response.Choices) == 0 {
			return fmt.Errorf("no choices returned")
		}
		output = response.Choices[0].Message.Content
		if aiChat.options.verbose {
			log.Printf("subsequent output: %s", output)
		}
		idx = nextIdx
	}
	fmt.Println(output)
	return nil
}

// tokenLimitOfModel returns the maximum number of tokens allowed for a given model.
func tokenLimitOfModel(model string) int {
	switch model {
	case gogpt.GPT4, gogpt.GPT40314:
		return 8 * 1024
	case gogpt.GPT3Dot5Turbo16K, gogpt.GPT3Dot5Turbo16K0613:
		return 16 * 1024
	case gogpt.GPT432K, gogpt.GPT432K0314:
		return 32 * 1024
	case gogpt.O4Mini, gogpt.O4Mini20250416:
		return 128 * 1024
	case gogpt.O3, gogpt.O3Mini, gogpt.O320250416, gogpt.O3Mini20250131:
		return 8 * 1024
	default:
		return 4 * 1024
	}
}

func hasBetaLimitations(model string) bool {
	switch model {
	case gogpt.O4Mini, gogpt.O4Mini20250416:
		return true
	default:
		return false
	}
}

func applyModelSpecificLimitations(request *gogpt.ChatCompletionRequest, verbose bool) bool {
	if !hasBetaLimitations(request.Model) {
		return false
	}

	if verbose {
		log.Printf("Applying beta limitations for model %s", request.Model)
	}

	request.Temperature = 1
	request.TopP = 1
	request.N = 1
	request.PresencePenalty = 0
	request.FrequencyPenalty = 0

	return true
}

func main() {
	var temperature float32 = 0.5
	var maxTokens = 0
	var verbose = false
	var listPrompts = false
	var nonStreaming = false
	var split = false
	var model = ""
	var saveHistory = false
	var loadHistory = ""
	var listHistory = false
	var deleteHistory = ""
	
	getopt.FlagLong(&temperature, "temperature", 't', "temperature")
	getopt.FlagLong(&maxTokens, "max-tokens", 0, "max tokens, 0 to use default")
	getopt.FlagLong(&verbose, "verbose", 'v', "verbose output")
	getopt.FlagLong(&listPrompts, "list-prompts", 'l', "list prompts")
	getopt.FlagLong(&nonStreaming, "non-streaming", 0, "non streaming mode")
	getopt.FlagLong(&split, "split", 0, "split input")
	getopt.FlagLong(&model, "model", 'm', "model")
	getopt.FlagLong(&saveHistory, "save", 0, "save conversation history")
	getopt.FlagLong(&loadHistory, "load", 0, "load conversation history by ID")
	getopt.FlagLong(&listHistory, "list-history", 0, "list saved conversations")
	getopt.FlagLong(&deleteHistory, "delete", 0, "delete conversation history by ID")
	getopt.Parse()

	if listPrompts {
		if err := ListPrompts(); err != nil {
			log.Fatal(err)
		}
		return
	}
	
	if listHistory {
		conversations, err := ListConversations()
		if err != nil {
			log.Fatal(err)
		}
		if len(conversations) == 0 {
			fmt.Println("No saved conversations.")
		} else {
			fmt.Println("Saved conversations:")
			for _, conv := range conversations {
				fmt.Printf("- %s: %s (%s)\n", conv.ID, conv.Title, conv.UpdatedAt.Format(time.RFC3339))
			}
		}
		return
	}
	
	if deleteHistory != "" {
		if err := DeleteConversation(deleteHistory); err != nil {
			log.Fatalf("Failed to delete conversation: %v", err)
		}
		fmt.Println("Conversation deleted.")
		return
	}

	openaiAPIKey, err := ReadOpenAIAPIKey()
	if err != nil {
		log.Fatal(err)
	}

	config, err := ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// if model is not specified, use the default model from the config file
	if model == "" {
		if config.Model == "" {
			model = gogpt.O4Mini
		} else {
			model = config.Model
		}
	}

	options := chatOptions{
		model:        model,
		temperature:  temperature,
		maxTokens:    maxTokens,
		nonStreaming: nonStreaming,
		verbose:      verbose,
		saveHistory:  saveHistory,
		loadHistory:  loadHistory,
		listHistory:  listHistory,
		deleteHistory: deleteHistory,
	}
	if verbose {
		log.Printf("options: %+v", options)
	}
	encoder, err := tokenizer.NewEncoder()
	if err != nil {
		log.Fatal(err)
	}
	
	aiChat := AIChat{
		client:  gogpt.NewClient(openaiAPIKey),
		encoder: encoder,
		options: options,
	}
	
	if loadHistory != "" {
		conversation, err := LoadConversation(loadHistory)
		if err != nil {
			log.Fatalf("Failed to load conversation: %v", err)
		}
		aiChat.conversation = conversation
		fmt.Printf("Loaded conversation: %s\n", conversation.Title)
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

		if prompt.isFoldEnabled() {
			if err := aiChat.fold(prompt, input); err != nil {
				log.Fatal(err)
			}
			return
		}

		var messagesSlice [][]gogpt.ChatCompletionMessage
		tokenLimit := tokenLimitOfModel(aiChat.options.model)

		if split {
			messagesSlice, err = prompt.CreateMessagesWithSplit(aiChat.encoder, input, tokenLimit, aiChat.options.maxTokens, aiChat.options.verbose)
			if err != nil {
				log.Fatal(err)
			}
			if verbose {
				log.Printf("messages was split to %d parts", len(messagesSlice))

			}
		} else {
			messages := prompt.CreateMessages(input)
			if verbose {
				log.Printf("messages: %+v", messages)
			}
			messagesSlice = [][]gogpt.ChatCompletionMessage{messages}
		}

		maxTokens := firstNonZeroInt(aiChat.options.maxTokens, prompt.MaxTokens)
		if verbose {
			log.Printf("max tokens: %d", maxTokens)
		}

		for _, messages := range messagesSlice {

			request := gogpt.ChatCompletionRequest{
				Model:       model,
				Messages:    messages,
				Temperature: firstNonZeroFloat32(prompt.Temperature, aiChat.options.temperature),
				MaxTokens:   maxTokens,
			}

			cnt, err := CountTokens(mapSlice(messages, func(m gogpt.ChatCompletionMessage) string { return m.Content }))
			if err != nil {
				log.Fatal(err)
			}
			if verbose {
				log.Printf("total tokens %d", cnt)
			}
			if cnt+maxTokens > tokenLimit {
				log.Fatalf("total tokens %d exceeds %d", cnt, tokenLimit)
			}

			applyModelSpecificLimitations(&request, aiChat.options.verbose)
			
			if aiChat.options.nonStreaming {
				err = nonStreamCompletion(aiChat.client, request, os.Stdout)
			} else {
				err = streamCompletion(aiChat.client, request, os.Stdout, aiChat.options.verbose)
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
