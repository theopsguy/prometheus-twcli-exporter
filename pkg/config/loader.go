package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfigFromFile(config *Config, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return err
	}

	return nil
}
