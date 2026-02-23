package exporter_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/config"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/exporter"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/twcli"
)

type mockTWCli struct {
	controllers    []string
	controllerInfo twcli.ControllerInfo
	unitStatus     twcli.UnitStatus
	driveInfo      []twcli.DriveInfo
	sataSmartData  twcli.SATASmartData
	err            error
}

func (m *mockTWCli) GetControllers() ([]string, error) {
	return m.controllers, m.err
}

func (m *mockTWCli) GetControllerInfo(controller string) (twcli.ControllerInfo, error) {
	return m.controllerInfo, m.err
}

func (m *mockTWCli) GetUnitStatus(controller string) (twcli.UnitStatus, error) {
	return m.unitStatus, m.err
}

func (m *mockTWCli) GetDriveStatus(controller string) ([]twcli.DriveInfo, error) {
	return m.driveInfo, m.err
}

func (m *mockTWCli) GetSATASmartData(controller, device string) (twcli.SATASmartData, error) {
	return m.sataSmartData, m.err
}

type labelMap map[string]string

type metricResult struct {
	labels     labelMap
	value      float64
	metricType io_prometheus_client.MetricType
}

func readMetric(m prometheus.Metric) metricResult {
	pb := &io_prometheus_client.Metric{}
	m.Write(pb)
	labels := make(labelMap, len(pb.Label))
	for _, v := range pb.Label {
		labels[v.GetName()] = v.GetValue()
	}

	if pb.Gauge != nil {
		return metricResult{labels: labels, value: pb.GetGauge().GetValue(), metricType: io_prometheus_client.MetricType_GAUGE}
	}
	if pb.Counter != nil {
		return metricResult{labels: labels, value: pb.GetCounter().GetValue(), metricType: io_prometheus_client.MetricType_COUNTER}
	}
	if pb.Summary != nil {
		return metricResult{labels: labels, value: pb.GetSummary().GetSampleSum(), metricType: io_prometheus_client.MetricType_SUMMARY}
	}
	if pb.Untyped != nil {
		return metricResult{labels: labels, value: pb.GetUntyped().GetValue(), metricType: io_prometheus_client.MetricType_UNTYPED}
	}
	panic("Unsupported metric type")
}

func TestNewExporterExecNotFound(t *testing.T) {
	cfg := config.Config{
		Executable:    "/usr/sbin/tw-cli",
		CacheDuration: 120,
		Listen: config.ListenConfig{
			Address: "0.0.0.0",
			Port:    9400,
		},
	}

	if os.Getenv("FORK") == "1" {
		exporter.New(cfg)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestNewExporterExecNotFound")
	cmd.Env = append(os.Environ(), "FORK=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	assert.Equal(t, err.Error(), "exit status 1")
	assert.Contains(t, stderr.String(), "Error running command error=\"fork/exec /usr/sbin/tw-cli: no such file or directory\"")
}

func TestCollectControllerInfo(t *testing.T) {
	tests := []struct {
		name     string
		mockData mockTWCli
		expected metricResult
	}{
		{
			name: "Controller Info",
			mockData: mockTWCli{
				controllerInfo: twcli.ControllerInfo{
					Controller:      "/c4",
					AvailableMemory: "234881024",
					BiosVersion:     "BE9X 4.08.00.004",
					FirmwareVersion: "FE9X 4.10.00.027",
					Model:           "9650SE-4LPML",
					SerialNumber:    "L1234568912345",
				},
			},
			expected: metricResult{
				labels: labelMap{
					"available_memory": "234881024",
					"bios_version":     "BE9X 4.08.00.004",
					"controller":       "/c4",
					"firmware_version": "FE9X 4.10.00.027",
					"model":            "9650SE-4LPML",
					"serial_number":    "L1234568912345",
				},
				value:      1.0,
				metricType: io_prometheus_client.MetricType_GAUGE,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controllerInventory := []twcli.ControllerInventory{
				{
					Name: "/c4",
					Devices: []twcli.Device{
						{Name: "/c4/p0", Type: "SATA"},
					},
				},
			}

			collector := exporter.Collector{
				ControllerInventory: controllerInventory,
				TWCli:               &tt.mockData,
			}
			e := exporter.Exporter{Collector: &collector}
			ch := make(chan prometheus.Metric, 10)
			result := e.Collector.CollectControllerInfo(ch)
			close(ch)

			assert.True(t, result)

			for m := range ch {
				data := readMetric(m)
				assert.Equal(t, tt.expected.value, data.value)
				assert.Equal(t, tt.expected.labels, data.labels)
				assert.Equal(t, tt.expected.metricType, data.metricType)
			}
		})
	}
}

func TestCollectUnitStatus(t *testing.T) {
	tests := []struct {
		name     string
		mockData mockTWCli
		expected metricResult
	}{
		{
			name: "OK",
			mockData: mockTWCli{
				unitStatus: twcli.UnitStatus{
					Unit:  "u0",
					Type:  "RAID-5",
					State: "OK",
				},
			},
			expected: metricResult{
				labels: labelMap{
					"controller": "/c4",
					"unit":       "u0",
					"type":       "RAID-5",
					"state":      "OK",
				},
				value:      1.0,
				metricType: io_prometheus_client.MetricType_GAUGE,
			},
		},
		{
			name: "REBUILDING",
			mockData: mockTWCli{
				unitStatus: twcli.UnitStatus{
					Unit:            "u0",
					Type:            "RAID-5",
					State:           "REBUILDING",
					PercentComplete: 35.0,
				},
			},
			expected: metricResult{
				labels: labelMap{
					"controller": "/c4",
					"unit":       "u0",
					"type":       "RAID-5",
					"state":      "REBUILDING",
				},
				value:      0.0,
				metricType: io_prometheus_client.MetricType_GAUGE,
			},
		},
		{
			name: "VERIFYING",
			mockData: mockTWCli{
				unitStatus: twcli.UnitStatus{
					Unit:            "u0",
					Type:            "RAID-5",
					State:           "VERIFYING",
					PercentComplete: 14.0,
				},
			},
			expected: metricResult{
				labels: labelMap{
					"controller": "/c4",
					"unit":       "u0",
					"type":       "RAID-5",
					"state":      "VERIFYING",
				},
				value:      1.0,
				metricType: io_prometheus_client.MetricType_GAUGE,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controllerInventory := []twcli.ControllerInventory{
				{
					Name: "/c4",
					Devices: []twcli.Device{
						{Name: "/c4/p0", Type: "SATA"},
					},
				},
			}

			collector := exporter.Collector{
				ControllerInventory: controllerInventory,
				TWCli:               &tt.mockData,
			}
			e := exporter.Exporter{Collector: &collector}

			ch := make(chan prometheus.Metric, 10)
			result := e.Collector.CollectUnitStatus(ch)
			close(ch)

			assert.True(t, result)

			for m := range ch {
				data := readMetric(m)
				if len(data.labels) == 4 {
					assert.Equal(t, tt.expected.value, data.value)
					assert.Equal(t, tt.expected.labels, data.labels)
					assert.Equal(t, tt.expected.metricType, data.metricType)
				}
			}
		})
	}
}

var (
	driveOK0 = twcli.DriveInfo{
		Status: "OK",
		Unit:   "u0",
		Size:   "3991227208827",
		Type:   "SATA",
		Phy:    "0",
		Model:  "ST4000VN006-3CW104",
	}
	driveOK1 = twcli.DriveInfo{
		Status: "OK",
		Unit:   "u0",
		Size:   "3991227208827",
		Type:   "SATA",
		Phy:    "1",
		Model:  "TOSHIBA HDWG440",
	}
	driveDegraded1 = twcli.DriveInfo{
		Status: "DEGRADED",
		Unit:   "u0",
		Size:   "3991227208827",
		Type:   "SATA",
		Phy:    "1",
		Model:  "TOSHIBA HDWG440",
	}
)

func TestCollectDriveStatus(t *testing.T) {
	tests := []struct {
		name     string
		mockData mockTWCli
		expected []metricResult
	}{
		{
			name:     "OK",
			mockData: mockTWCli{driveInfo: []twcli.DriveInfo{driveOK0, driveOK1}},
			expected: []metricResult{
				{
					labels: labelMap{
						"status": "OK",
						"unit":   "u0",
						"size":   "3991227208827",
						"type":   "SATA",
						"phy":    "0",
						"model":  "ST4000VN006-3CW104",
					},
					value:      1,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
				{
					labels: labelMap{
						"status": "OK",
						"unit":   "u0",
						"size":   "3991227208827",
						"type":   "SATA",
						"phy":    "1",
						"model":  "TOSHIBA HDWG440",
					},
					value:      1,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
			},
		},
		{
			name:     "DEGRADED",
			mockData: mockTWCli{driveInfo: []twcli.DriveInfo{driveOK0, driveDegraded1}},
			expected: []metricResult{
				{
					labels: labelMap{
						"status": "OK",
						"unit":   "u0",
						"size":   "3991227208827",
						"type":   "SATA",
						"phy":    "0",
						"model":  "ST4000VN006-3CW104",
					},
					value:      1,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
				{
					labels: labelMap{
						"status": "DEGRADED",
						"unit":   "u0",
						"size":   "3991227208827",
						"type":   "SATA",
						"phy":    "1",
						"model":  "TOSHIBA HDWG440",
					},
					value:      0,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controllerInventory := []twcli.ControllerInventory{
				{
					Name: "/c4",
					Devices: []twcli.Device{
						{Name: "/c4/p0", Type: "SATA"},
						{Name: "/c4/p1", Type: "SATA"},
					},
				},
			}

			collector := exporter.Collector{
				ControllerInventory: controllerInventory,
				TWCli:               &tt.mockData,
			}
			e := exporter.Exporter{Collector: &collector}

			ch := make(chan prometheus.Metric, 10)
			result := e.Collector.CollectDriveStatus(ch)
			close(ch)

			assert.True(t, result)

			var actual []metricResult
			for m := range ch {
				actual = append(actual, readMetric(m))
			}

			assert.Len(t, actual, len(tt.expected))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.value, actual[i].value)
				assert.Equal(t, expected.labels, actual[i].labels)
				assert.Equal(t, expected.metricType, actual[i].metricType)
			}
		})
	}
}

func TestCollectDriveSmartData(t *testing.T) {
	smartLabelMap := labelMap{
		"status":        "OK",
		"model":         "ST4000VN006-3CW104",
		"serial":        "AA12345",
		"spindle_speed": "5400",
		"unit":          "u0",
	}

	tests := []struct {
		name     string
		mockData mockTWCli
		expected []metricResult
	}{
		{
			name: "HealthyDrive",
			mockData: mockTWCli{
				sataSmartData: twcli.SATASmartData{
					Controller:         "/c4",
					Device:             "/c4/p0",
					Status:             "OK",
					Model:              "ST4000VN006-3CW104",
					Serial:             "AA12345",
					Unit:               "u0",
					ReallocatedSectors: "0",
					PowerOnHours:       "2355",
					Temperature:        "31",
					SpindleSpeed:       "5400",
				},
			},
			expected: []metricResult{
				{
					labels:     smartLabelMap,
					value:      0,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
				{
					labels:     smartLabelMap,
					value:      2355,
					metricType: io_prometheus_client.MetricType_COUNTER,
				},
				{
					labels:     smartLabelMap,
					value:      31,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
			},
		},
		{
			name: "ReallocatedSectors",
			mockData: mockTWCli{
				sataSmartData: twcli.SATASmartData{
					Controller:         "/c4",
					Device:             "/c4/p0",
					Status:             "OK",
					Model:              "ST4000VN006-3CW104",
					Serial:             "AA12345",
					Unit:               "u0",
					ReallocatedSectors: "1039",
					PowerOnHours:       "2355",
					Temperature:        "31",
					SpindleSpeed:       "5400",
				},
			},
			expected: []metricResult{
				{
					labels:     smartLabelMap,
					value:      1039,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
				{
					labels:     smartLabelMap,
					value:      2355,
					metricType: io_prometheus_client.MetricType_COUNTER,
				},
				{
					labels:     smartLabelMap,
					value:      31,
					metricType: io_prometheus_client.MetricType_GAUGE,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controllerInventory := []twcli.ControllerInventory{
				{
					Name: "/c4",
					Devices: []twcli.Device{
						{Name: "/c4/p0", Type: "SATA"},
					},
				},
			}

			collector := exporter.Collector{
				ControllerInventory: controllerInventory,
				TWCli:               &tt.mockData,
			}
			e := exporter.Exporter{Collector: &collector}

			ch := make(chan prometheus.Metric, 10)
			result := e.Collector.CollectDriveSmartData(ch)
			close(ch)

			assert.True(t, result)

			var actual []metricResult
			for m := range ch {
				actual = append(actual, readMetric(m))
			}

			assert.Len(t, actual, len(tt.expected))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.value, actual[i].value)
				assert.Equal(t, expected.labels, actual[i].labels)
				assert.Equal(t, expected.metricType, actual[i].metricType)
			}
		})
	}
}

type mockCollector struct {
	ctrlOK, unitOK, driveOK, smartOK bool
}

func (m *mockCollector) CollectControllerInfo(ch chan<- prometheus.Metric) bool {
	return m.ctrlOK
}

func (m *mockCollector) CollectUnitStatus(ch chan<- prometheus.Metric) bool {
	return m.unitOK
}

func (m *mockCollector) CollectDriveStatus(ch chan<- prometheus.Metric) bool {
	return m.driveOK
}

func (m *mockCollector) CollectDriveSmartData(ch chan<- prometheus.Metric) bool {
	return m.smartOK
}

func TestExporterCollectStatus(t *testing.T) {
	tests := []struct {
		name     string
		mockData mockCollector
		expected float64
	}{
		{
			name: "OK",
			mockData: mockCollector{
				ctrlOK:  true,
				unitOK:  true,
				driveOK: true,
				smartOK: true,
			},
			expected: 1.0,
		},
		{
			name: "FAIL",
			mockData: mockCollector{
				ctrlOK:  false,
				unitOK:  true,
				driveOK: true,
				smartOK: true,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan prometheus.Metric, 2)
			e := &exporter.Exporter{
				Collector: &tt.mockData,
			}
			e.Collect(ch)
			close(ch)

			assert.Len(t, ch, 2)
			for metric := range ch {
				desc := metric.Desc().String()
				if strings.Contains(desc, "tw_cli_scrape_collector_success") {
					data := readMetric(metric)
					assert.Equal(t, tt.expected, data.value)
				}
			}
		})
	}
}
