package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogpt "github.com/sashabaranov/go-openai"
)

func TestNewConversation(t *testing.T) {
	title := "Test Conversation"
	model := "gpt-3.5-turbo"
	
	conversation := NewConversation(title, model)
	
	if conversation.ID == "" {
		t.Error("Expected conversation ID to be generated")
	}
	
	if conversation.Title != title {
		t.Errorf("Expected title to be %q, got %q", title, conversation.Title)
	}
	
	if conversation.Model != model {
		t.Errorf("Expected model to be %q, got %q", model, conversation.Model)
	}
	
	if len(conversation.Messages) != 0 {
		t.Errorf("Expected messages to be empty, got %d messages", len(conversation.Messages))
	}
	
	if conversation.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	
	if conversation.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestAddMessage(t *testing.T) {
	conversation := NewConversation("Test", "gpt-3.5-turbo")
	
	conversation.AddMessage("user", "Hello")
	
	if len(conversation.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(conversation.Messages))
	}
	
	msg := conversation.Messages[0]
	if msg.Role != "user" {
		t.Errorf("Expected role to be 'user', got %q", msg.Role)
	}
	
	if msg.Content != "Hello" {
		t.Errorf("Expected content to be 'Hello', got %q", msg.Content)
	}
	
	if msg.Time.IsZero() {
		t.Error("Expected message time to be set")
	}
	
	if conversation.UpdatedAt.Equal(conversation.CreatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}
}

func TestToGPTMessages(t *testing.T) {
	conversation := NewConversation("Test", "gpt-3.5-turbo")
	conversation.AddMessage("user", "Hello")
	conversation.AddMessage("assistant", "Hi there!")
	
	messages := conversation.ToGPTMessages()
	
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}
	
	if messages[0].Role != "user" || messages[0].Content != "Hello" {
		t.Errorf("Expected first message to be user:Hello, got %s:%s", messages[0].Role, messages[0].Content)
	}
	
	if messages[1].Role != "assistant" || messages[1].Content != "Hi there!" {
		t.Errorf("Expected second message to be assistant:Hi there!, got %s:%s", messages[1].Role, messages[1].Content)
	}
}

func TestFromGPTMessages(t *testing.T) {
	conversation := NewConversation("Test", "gpt-3.5-turbo")
	
	gptMessages := []gogpt.ChatCompletionMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}
	
	conversation.FromGPTMessages(gptMessages)
	
	if len(conversation.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(conversation.Messages))
	}
	
	if conversation.Messages[0].Role != "user" || conversation.Messages[0].Content != "Hello" {
		t.Errorf("Expected first message to be user:Hello, got %s:%s", 
			conversation.Messages[0].Role, conversation.Messages[0].Content)
	}
	
	if conversation.Messages[1].Role != "assistant" || conversation.Messages[1].Content != "Hi there!" {
		t.Errorf("Expected second message to be assistant:Hi there!, got %s:%s", 
			conversation.Messages[1].Role, conversation.Messages[1].Content)
	}
}

func TestGetConversationTitle(t *testing.T) {
	title := GetConversationTitle([]gogpt.ChatCompletionMessage{})
	if title != "New Conversation" {
		t.Errorf("Expected 'New Conversation', got %q", title)
	}
	
	messages := []gogpt.ChatCompletionMessage{
		{Role: "user", Content: "This is a test message"},
	}
	title = GetConversationTitle(messages)
	if title != "This is a test message" {
		t.Errorf("Expected 'This is a test message', got %q", title)
	}
	
	messages = []gogpt.ChatCompletionMessage{
		{Role: "user", Content: "This is a very long test message that should be truncated"},
	}
	title = GetConversationTitle(messages)
	if title != "This is a very long test mess..." {
		t.Errorf("Expected 'This is a very long test mess...', got %q", title)
	}
	
	messages = []gogpt.ChatCompletionMessage{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "User message"},
	}
	title = GetConversationTitle(messages)
	if title != "User message" {
		t.Errorf("Expected 'User message', got %q", title)
	}
}

func TestSaveLoadConversation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "aichat-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	origGetHistoryDir := GetHistoryDir
	defer func() { GetHistoryDir = origGetHistoryDir }()
	
	GetHistoryDir = func() (string, error) {
		return tempDir, nil
	}
	
	conversation := NewConversation("Test Conversation", "gpt-3.5-turbo")
	conversation.AddMessage("user", "Hello")
	conversation.AddMessage("assistant", "Hi there!")
	
	if err := SaveConversation(conversation); err != nil {
		t.Fatalf("Failed to save conversation: %v", err)
	}
	
	filePath := filepath.Join(tempDir, conversation.ID+".yml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist: %s", filePath)
	}
	
	loaded, err := LoadConversation(conversation.ID)
	if err != nil {
		t.Fatalf("Failed to load conversation: %v", err)
	}
	
	if loaded.ID != conversation.ID {
		t.Errorf("Expected ID to be %q, got %q", conversation.ID, loaded.ID)
	}
	
	if loaded.Title != conversation.Title {
		t.Errorf("Expected Title to be %q, got %q", conversation.Title, loaded.Title)
	}
	
	if len(loaded.Messages) != len(conversation.Messages) {
		t.Fatalf("Expected %d messages, got %d", len(conversation.Messages), len(loaded.Messages))
	}
	
	for i, msg := range loaded.Messages {
		if msg.Role != conversation.Messages[i].Role {
			t.Errorf("Message %d: Expected Role to be %q, got %q", i, conversation.Messages[i].Role, msg.Role)
		}
		
		if msg.Content != conversation.Messages[i].Content {
			t.Errorf("Message %d: Expected Content to be %q, got %q", i, conversation.Messages[i].Content, msg.Content)
		}
	}
}

func TestListConversations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "aichat-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	origGetHistoryDir := GetHistoryDir
	defer func() { GetHistoryDir = origGetHistoryDir }()
	
	GetHistoryDir = func() (string, error) {
		return tempDir, nil
	}
	
	conversations, err := ListConversations()
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}
	
	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(conversations))
	}
	
	for i := 0; i < 3; i++ {
		conversation := NewConversation(fmt.Sprintf("Test %d", i), "gpt-3.5-turbo")
		conversation.AddMessage("user", fmt.Sprintf("Message %d", i))
		
		if err := SaveConversation(conversation); err != nil {
			t.Fatalf("Failed to save conversation: %v", err)
		}
		
		time.Sleep(10 * time.Millisecond)
	}
	
	conversations, err = ListConversations()
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}
	
	if len(conversations) != 3 {
		t.Errorf("Expected 3 conversations, got %d", len(conversations))
	}
}

func TestDeleteConversation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "aichat-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	origGetHistoryDir := GetHistoryDir
	defer func() { GetHistoryDir = origGetHistoryDir }()
	
	GetHistoryDir = func() (string, error) {
		return tempDir, nil
	}
	
	conversation := NewConversation("Test", "gpt-3.5-turbo")
	conversation.AddMessage("user", "Hello")
	
	if err := SaveConversation(conversation); err != nil {
		t.Fatalf("Failed to save conversation: %v", err)
	}
	
	if err := DeleteConversation(conversation.ID); err != nil {
		t.Fatalf("Failed to delete conversation: %v", err)
	}
	
	filePath := filepath.Join(tempDir, conversation.ID+".yml")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected file to be deleted: %s", filePath)
	}
	
	_, err = LoadConversation(conversation.ID)
	if err == nil {
		t.Error("Expected error when loading deleted conversation")
	}
}
