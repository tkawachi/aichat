package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	gogpt "github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

type ChatMessage struct {
	Role    string    `yaml:"role"`
	Content string    `yaml:"content"`
	Time    time.Time `yaml:"time"`
}

type Conversation struct {
	ID        string        `yaml:"id"`
	Title     string        `yaml:"title"`
	Messages  []ChatMessage `yaml:"messages"`
	CreatedAt time.Time     `yaml:"created_at"`
	UpdatedAt time.Time     `yaml:"updated_at"`
	Model     string        `yaml:"model"`
}

func NewConversation(title, model string) *Conversation {
	return &Conversation{
		ID:        uuid.New().String(),
		Title:     title,
		Messages:  []ChatMessage{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Model:     model,
	}
}

func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, ChatMessage{
		Role:    role,
		Content: content,
		Time:    time.Now(),
	})
	c.UpdatedAt = time.Now()
}

func (c *Conversation) ToGPTMessages() []gogpt.ChatCompletionMessage {
	messages := make([]gogpt.ChatCompletionMessage, len(c.Messages))
	for i, msg := range c.Messages {
		messages[i] = gogpt.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return messages
}

func (c *Conversation) FromGPTMessages(messages []gogpt.ChatCompletionMessage) {
	c.Messages = make([]ChatMessage, len(messages))
	for i, msg := range messages {
		c.Messages[i] = ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Time:    time.Now(),
		}
	}
	c.UpdatedAt = time.Now()
}

type GetHistoryDirFunc func() (string, error)

var GetHistoryDir GetHistoryDirFunc = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	historyDir := filepath.Join(homeDir, ".aichat", "history")
	
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		return "", err
	}
	
	return historyDir, nil
}

func SaveConversation(conversation *Conversation) error {
	historyDir, err := GetHistoryDir()
	if err != nil {
		return err
	}
	
	conversation.UpdatedAt = time.Now()
	
	filePath := filepath.Join(historyDir, fmt.Sprintf("%s.yml", conversation.ID))
	
	data, err := yaml.Marshal(conversation)
	if err != nil {
		return err
	}
	
	return os.WriteFile(filePath, data, 0600)
}

func LoadConversation(id string) (*Conversation, error) {
	historyDir, err := GetHistoryDir()
	if err != nil {
		return nil, err
	}
	
	filePath := filepath.Join(historyDir, fmt.Sprintf("%s.yml", id))
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	conversation := &Conversation{}
	if err := yaml.Unmarshal(data, conversation); err != nil {
		return nil, err
	}
	
	return conversation, nil
}

func ListConversations() ([]*Conversation, error) {
	historyDir, err := GetHistoryDir()
	if err != nil {
		return nil, err
	}
	
	files, err := os.ReadDir(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Conversation{}, nil
		}
		return nil, err
	}
	
	conversations := []*Conversation{}
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		if !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}
		
		id := filepath.Base(file.Name())
		id = id[:len(id)-4] // Remove .yml extension
		
		conversation, err := LoadConversation(id)
		if err != nil {
			continue // Skip files that can't be loaded
		}
		
		conversations = append(conversations, conversation)
	}
	
	return conversations, nil
}

func DeleteConversation(id string) error {
	historyDir, err := GetHistoryDir()
	if err != nil {
		return err
	}
	
	filePath := filepath.Join(historyDir, fmt.Sprintf("%s.yml", id))
	
	return os.Remove(filePath)
}

func GetConversationTitle(messages []gogpt.ChatCompletionMessage) string {
	if len(messages) == 0 {
		return "New Conversation"
	}
	
	for _, msg := range messages {
		if msg.Role == gogpt.ChatMessageRoleUser {
			title := msg.Content
			if len(title) > 30 {
				title = title[:29] + "..."
			}
			return title
		}
	}
	
	return "New Conversation"
}
