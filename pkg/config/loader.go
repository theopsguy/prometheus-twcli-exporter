package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type StartupFlags struct {
	ConfigFile    string
	WebConfigFile string
	Version       bool
}

type Config struct {
	Listen        ListenConfig
	CacheDuration int
	Executable    string
	Log           LogConfig
	MetricsPath   string
}

type ListenConfig struct {
	Port    int
	Address string
}

type LogConfig struct {
	Level  string
	Format string
}

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
