package main

import (
	"os"
	"path/filepath"
)

type Config struct {
	Model string `yaml:"model"`
}

func ReadConfig() (*Config, error) {
	config := &Config{}
	// try to read from ~/.aichat/config.yml
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(homedir, ".aichat", "config.yml")

	if err := ReadYamlFromFile(path, config); err != nil {
		// if the file does not exist, return an empty config
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}
	return config, nil
}
