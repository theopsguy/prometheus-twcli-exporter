package config

type StartupFlags struct {
	ConfigFile string
	Version    bool
}

type Config struct {
	Listen        ListenConfig
	CacheDuration int
	Executable    string
}

type ListenConfig struct {
	Port    int
	Address string
}
