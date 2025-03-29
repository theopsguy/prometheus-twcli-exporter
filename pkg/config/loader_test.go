package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertConfigContents(t *testing.T, cfg Config) {
	assert.Equal(t, "0.0.0.0", cfg.Listen.Address)
	assert.Equal(t, 9400, cfg.Listen.Port)
	assert.Equal(t, "/usr/sbin/tw-cli", cfg.Executable)
	assert.Equal(t, 120, cfg.CacheDuration)
}

func TestLoadsYAMLConfigFile(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	configFile := "testdata/config.yaml"
	err := LoadConfigFromFile(&cfg, configFile)
	assert.Nil(t, err, "unexpected error: %v", err)
	assertConfigContents(t, cfg)
}
