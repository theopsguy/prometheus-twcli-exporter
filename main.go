package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/config"
	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/exporter"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

const (
	exporterName = "prometheus_twcli_exporter"
)

func main() {
	var opts config.StartupFlags

	flag.StringVar(&opts.ConfigFile, "config-file", "", "Configuration file to read from")
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
	}
	loadConfig(&opts, &cfg)
	setupLogger(&cfg)

	listenAddr := fmt.Sprintf("%s:%d", cfg.Listen.Address, cfg.Listen.Port)
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

	http.Handle("/metrics",
		promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer,
			promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{},
			),
		),
	)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>tw-cli Exporter</title></head>
             <body>
             <h1>tw-cli Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </dl>
             <h2>Build</h2>
             <pre>` + version.Info() + ` ` + version.BuildContext() + `</pre>
             </body>
             </html>`))
	})
	http.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		slog.Error("Server failed to start", "addr", listenAddr, "error", err)
	}
}

func setupLogger(cfg *config.Config) {
	var handler slog.Handler
	level := new(slog.LevelVar)
	switch cfg.Log.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
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
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
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
