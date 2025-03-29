package main

import (
	"flag"
	"fmt"
	"log"
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
	exporterName = "prometheus-twcli-exporter"
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
	}
	loadConfig(&opts, &cfg)

	listenAddr := fmt.Sprintf("%s:%d", cfg.Listen.Address, cfg.Listen.Port)
	log.Printf("Running HTTP server on address %s\n", listenAddr)

	twcliExporter, err := exporter.New(cfg)
	if err != nil {
		log.Fatal("Error creating the exporter")
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
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func loadConfig(opts *config.StartupFlags, cfg *config.Config) {
	if opts.ConfigFile != "" {
		log.Printf("Loading configuration file: %s\n", opts.ConfigFile)
		err := config.LoadConfigFromFile(cfg, opts.ConfigFile)
		if err != nil {
			log.Fatal(err)
		}
	}
}
