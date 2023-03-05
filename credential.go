package main

import (
	"log"
	"os"
	"path/filepath"
)

type Credentials struct {
	OpenAIAPIKey string `yaml:"openai_api_key"`
}

func ReadOpenAIAPIKey() (string, error) {
	// first try to read from env
	key, found := os.LookupEnv("OPENAI_API_KEY")
	if found {
		return key, nil
	}
	// then try to read from ~/.aichat/credentials.yml
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(homedir, ".aichat", "credentials.yml")
	// check permission and warn if its too open
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&0077 != 0 {
		log.Printf("WARN: credentials file %s has too open permission\n", path)
	}
	credentials := &Credentials{}
	if err := ReadYamlFromFile(path, credentials); err != nil {
		return "", err
	}
	return credentials.OpenAIAPIKey, nil
}
