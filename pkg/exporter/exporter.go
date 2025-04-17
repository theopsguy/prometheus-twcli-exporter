package exporter

import (
	"log"
	"os"
	"slices"
	"time"

	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/config"
	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/shell"
	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/twcli"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "tw_cli"
)

type MetricsCollector interface {
	CollectControllerDetails(ch chan<- prometheus.Metric) bool
	CollectUnitStatus(ch chan<- prometheus.Metric) bool
	CollectDriveStatus(ch chan<- prometheus.Metric) bool
}

type Exporter struct {
	Controllers []string
	TWCli       twcli.TWCli
	Collector   MetricsCollector
}

var (
	controllerInfo = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "controller", "info"),
		"Controller information",
		[]string{"controller", "model", "available_memory", "firmware_version", "bios_version", "serial_number"}, nil,
	)
	unitStatusDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "unit", "status"),
		"Unit Status",
		[]string{"controller", "unit", "type", "state"}, nil,
	)
	percentCompleteDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "unit", "percent_complete"),
		"Report percent complete if unit is rebuilding or verifying",
		[]string{"controller", "unit", "state"}, nil,
	)
	driveStatusDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "drive", "status"),
		"Drive Status",
		[]string{"status", "unit", "size", "type", "phy", "model"}, nil,
	)
	scrapeDuration = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"Number of seconds taken to scrape metrics",
		[]string{}, nil,
	)
	scrapeSuccess = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"Indicates if any failures occured during scrape",
		[]string{}, nil,
	)
)

func New(cfg config.Config) (*Exporter, error) {
	shell := shell.LocalShell{}
	twcli := twcli.New(cfg.CacheDuration, cfg.Executable, shell)
	controllers, err := twcli.GetControllers()
	if err != nil {
		log.Fatal("Error querying controllers")
		os.Exit(1)
	}

	return &Exporter{
		Controllers: controllers,
		TWCli:       *twcli,
	}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDuration
	ch <- scrapeSuccess
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	var success float64 = 1

	ok := e.Collector.CollectControllerDetails(ch)
	ok = e.Collector.CollectUnitStatus(ch) && ok
	ok = e.Collector.CollectDriveStatus(ch) && ok

	if !ok {
		success = 0
	}

	duration := time.Since(start)
	ch <- prometheus.MustNewConstMetric(scrapeDuration, prometheus.GaugeValue, duration.Seconds())
	ch <- prometheus.MustNewConstMetric(scrapeSuccess, prometheus.GaugeValue, success)

}

func (e *Exporter) CollectControllerDetails(ch chan<- prometheus.Metric) bool {

	for _, controller := range e.Controllers {
		labels, err := e.TWCli.GetControllerInfo(controller)
		if err != nil {
			return false
		}

		ch <- prometheus.MustNewConstMetric(
			controllerInfo, prometheus.GaugeValue, 1.0, labels...,
		)
	}

	return true
}

func (e *Exporter) CollectUnitStatus(ch chan<- prometheus.Metric) bool {
	okStates := []string{"OK", "VERIFYING"}
	percentStates := []string{"VERIFYING", "REBUILDING"}
	var statusGaugeValue float64 = 0

	for _, controller := range e.Controllers {
		unit, unitType, unitStatus, percentComplete, err := e.TWCli.GetUnitStatus(controller)
		if err != nil {
			return false
		}

		if slices.Contains(okStates, unitStatus) {
			statusGaugeValue = 1
		}

		ch <- prometheus.MustNewConstMetric(
			unitStatusDesc, prometheus.GaugeValue, statusGaugeValue, controller, unit, unitType, unitStatus,
		)

		if slices.Contains(percentStates, unitStatus) {
			ch <- prometheus.MustNewConstMetric(
				percentCompleteDesc, prometheus.GaugeValue, float64(percentComplete), controller, unit, unitStatus,
			)
		}
	}

	return true
}

func (e *Exporter) CollectDriveStatus(ch chan<- prometheus.Metric) bool {
	var statusGaugeValue float64 = 0

	for _, controller := range e.Controllers {
		drives, err := e.TWCli.GetDriveStatus(controller)
		if err != nil {
			return false
		}

		for _, drive := range drives {
			if drive.Status == "OK" {
				statusGaugeValue = 1
			}

			ch <- prometheus.MustNewConstMetric(
				driveStatusDesc, prometheus.GaugeValue, statusGaugeValue, drive.Status, drive.Unit, drive.Size, drive.Type, drive.Phy, drive.Model,
			)
		}
	}

	return true
}
