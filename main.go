package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/config"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/exporter"
)

const (
	exporterName = "prometheus_twcli_exporter"
)

func main() {
	var opts config.StartupFlags

	flag.StringVar(&opts.ConfigFile, "config-file", "", "Configuration file to read from")
	flag.StringVar(&opts.WebConfigFile, "web-config-file", "", "Use to enable TLS, HTTP Basic Auth")
	flag.BoolVar(&opts.Version, "version", false, "Print version information")
	flag.Parse()

	if opts.Version {
		fmt.Println(version.Print(exporterName))
		os.Exit(0)
	}

	cfg := config.Config{
		Executable:    "/usr/sbin/tw-cli",
		CacheDuration: 120,
		Listen: config.ListenConfig{
			Address: "0.0.0.0",
			Port:    9400,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "text",
		},
		MetricsPath: "/metrics",
	}
	loadConfig(&opts, &cfg)
	logger := setupLogger(&cfg)

	slog.Info("Starting twcli_exporter", "version", version.Info())
	twcliExporter, err := exporter.New(cfg)
	if err != nil {
		slog.Error("Error creating exporter", "error", err)
		os.Exit(1)
	}

	prometheus.MustRegister(
		versioncollector.NewCollector(exporterName),
		twcliExporter,
	)

	http.Handle(cfg.MetricsPath, promhttp.Handler())
	if cfg.MetricsPath != "/" {
		landingConfig := web.LandingConfig{
			Name:        "TWCLI Exporter",
			Description: "Prometheus exporter for 3ware RAID cards.",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: cfg.MetricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			slog.Error("Error creating landing page", "error", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.Listen.Address, cfg.Listen.Port)
	flags := web.FlagConfig{
		WebListenAddresses: &[]string{listenAddr},
		WebConfigFile:      &opts.WebConfigFile,
	}
	server := &http.Server{}
	if err := web.ListenAndServe(server, &flags, logger); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func setupLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler
	level := new(slog.LevelVar)
	switch cfg.Log.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})

	}

	switch cfg.Log.Level {
	case "debug":
		level.Set(slog.LevelDebug)
	case "info":
		level.Set(slog.LevelInfo)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func loadConfig(opts *config.StartupFlags, cfg *config.Config) {
	if opts.ConfigFile != "" {
		slog.Info("Loading configuration", "config_file", opts.ConfigFile)
		err := config.LoadConfigFromFile(cfg, opts.ConfigFile)
		if err != nil {
			slog.Error("Error loading configuration", "file", opts.ConfigFile, "error", err)
		}
	}
}
