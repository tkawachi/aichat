package main

import (
	"fmt"
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
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("OpenAI API credentials not found. To use aichat, you need to set up your API key using one of these methods:\n\n" +
				"1. Create a credentials file at: %s with the following content:\n" +
				"   ```yaml\n" +
				"   openai_api_key: YOUR_API_KEY\n" +
				"   ```\n\n" +
				"2. Or set the OPENAI_API_KEY environment variable:\n" +
				"   export OPENAI_API_KEY=your_api_key\n\n" +
				"See README.md for more information.", path)
		}
		return "", err
	}
	// check permission and warn if its too open
	if info.Mode()&0077 != 0 {
		log.Printf("WARN: credentials file %s has too open permission\n", path)
	}
	credentials := &Credentials{}
	if err := ReadYamlFromFile(path, credentials); err != nil {
		return "", err
	}
	return credentials.OpenAIAPIKey, nil
}
