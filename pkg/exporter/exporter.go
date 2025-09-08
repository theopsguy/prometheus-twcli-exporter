package exporter

import (
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/config"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/shell"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/twcli"
)

const (
	namespace = "tw_cli"
)

type MetricsCollector interface {
	CollectControllerDetails(ch chan<- prometheus.Metric) bool
	CollectUnitStatus(ch chan<- prometheus.Metric) bool
	CollectDriveStatus(ch chan<- prometheus.Metric) bool
	CollectDriveSmartData(ch chan<- prometheus.Metric) bool
}

type Collector struct {
	ControllerData []twcli.ControllerInfo
	TWCli          twcli.TWCli
}

type Exporter struct {
	Collector MetricsCollector
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
	driveReallocatedSectorsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "drive", "reallocated_sectors"),
		"Drive Reallocated Sectors",
		[]string{"status", "model", "serial", "spindle_speed", "unit"}, nil,
	)
	drivePowerOnHoursDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "drive", "power_on_hours"),
		"Drive Power On Hours",
		[]string{"status", "model", "serial", "spindle_speed", "unit"}, nil,
	)
	parseErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "drive_smart_parse_errors_total",
			Help: "Total number of parse errors when reading SMART data fields.",
		},
		[]string{"field"},
	)
	driveTemperatureDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "drive", "temperature"),
		"Drive Temperature",
		[]string{"status", "model", "serial", "spindle_speed", "unit"}, nil,
	)
	scrapeDuration = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"Number of seconds taken to scrape metrics",
		[]string{}, nil,
	)
	scrapeSuccess = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"Indicates if any failures occurred during scrape",
		[]string{}, nil,
	)
)

func New(cfg config.Config) (*Exporter, error) {
	shell := shell.LocalShell{}
	t := twcli.New(cfg.CacheDuration, cfg.Executable, shell)

	controllers, err := t.GetControllers()
	if err != nil {
		slog.Error("Error querying controllers", "error", err)
		os.Exit(1)
	}

	var controllerData []twcli.ControllerInfo

	for _, controller := range controllers {
		devices, err := t.GetDevices(controller)
		if err != nil {
			slog.Error("Error getting devices", "controller", controller, "error", err)
		}
		controllerData = append(controllerData, twcli.ControllerInfo{
			Name:    controller,
			Devices: devices,
		})

	}

	collector := &Collector{
		ControllerData: controllerData,
		TWCli:          *t,
	}

	return &Exporter{
		Collector: collector,
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
	ok = e.Collector.CollectDriveSmartData(ch) && ok

	if !ok {
		success = 0
	}

	duration := time.Since(start)
	ch <- prometheus.MustNewConstMetric(scrapeDuration, prometheus.GaugeValue, duration.Seconds())
	ch <- prometheus.MustNewConstMetric(scrapeSuccess, prometheus.GaugeValue, success)

}

func (c *Collector) CollectControllerDetails(ch chan<- prometheus.Metric) bool {

	for _, controllerData := range c.ControllerData {
		labels, err := c.TWCli.GetControllerInfo(controllerData.Name)
		if err != nil {
			return false
		}

		ch <- prometheus.MustNewConstMetric(
			controllerInfo, prometheus.GaugeValue, 1.0, labels...,
		)
	}

	return true
}

func (c *Collector) CollectUnitStatus(ch chan<- prometheus.Metric) bool {
	okStates := []string{"OK", "VERIFYING"}
	percentStates := []string{"VERIFYING", "REBUILDING"}
	var statusGaugeValue float64 = 0

	for _, controllerData := range c.ControllerData {
		unit, unitType, unitStatus, percentComplete, err := c.TWCli.GetUnitStatus(controllerData.Name)
		if err != nil {
			return false
		}

		if slices.Contains(okStates, unitStatus) {
			statusGaugeValue = 1
		}

		ch <- prometheus.MustNewConstMetric(
			unitStatusDesc, prometheus.GaugeValue, statusGaugeValue, controllerData.Name, unit, unitType, unitStatus,
		)

		if slices.Contains(percentStates, unitStatus) {
			ch <- prometheus.MustNewConstMetric(
				percentCompleteDesc, prometheus.GaugeValue, float64(percentComplete), controllerData.Name, unit, unitStatus,
			)
		}
	}

	return true
}

func (c *Collector) CollectDriveStatus(ch chan<- prometheus.Metric) bool {
	for _, controllerData := range c.ControllerData {
		drives, err := c.TWCli.GetDriveStatus(controllerData.Name)
		if err != nil {
			return false
		}

		for _, drive := range drives {
			var statusGaugeValue float64 = 0

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

func (c *Collector) CollectDriveSmartData(ch chan<- prometheus.Metric) bool {
	for _, controller := range c.ControllerData {
		for _, device := range controller.Devices {
			switch device.Type {
			case "SATA":
				data, err := c.TWCli.GetSATASmartData(controller.Name, device.Name)
				if err != nil {
					slog.Error("Error getting SATA SMART data", "device", device.Name, "error", err)
					return false
				}
				c.emitSATAMetrics(data, ch)
			default:
				slog.Warn("Unsupported drive data type", "device", device.Name, "type", device.Type)
				return false
			}
		}
	}
	return true
}

func (c *Collector) emitSATAMetrics(data *twcli.SATASmartData, ch chan<- prometheus.Metric) {
	status := data.Status
	model := data.Model
	serial := data.Serial
	spindleSpeed := data.SpindleSpeed
	unit := data.Unit

	reallocatedSectorsFloat, ok := parseFloat(data.ReallocatedSectors, "ReallocatedSectors")
	if ok {
		ch <- prometheus.MustNewConstMetric(
			driveReallocatedSectorsDesc, prometheus.GaugeValue, reallocatedSectorsFloat, status, model, serial, spindleSpeed, unit,
		)
	}
	powerOnHoursFloat, ok := parseFloat(data.PowerOnHours, "PowerOnHours")
	if ok {
		ch <- prometheus.MustNewConstMetric(
			drivePowerOnHoursDesc, prometheus.CounterValue, powerOnHoursFloat, status, model, serial, spindleSpeed, unit,
		)
	}
	temperatureFloat, ok := parseFloat(data.Temperature, "Temperature")
	if ok {
		ch <- prometheus.MustNewConstMetric(
			driveTemperatureDesc, prometheus.GaugeValue, temperatureFloat, status, model, serial, spindleSpeed, unit,
		)
	}
}
