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
	"github.com/theopsguy/prometheus-twcli-exporter/internal/testutil"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/config"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/exporter"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/twcli"
)

type labelMap map[string]string

type metricResult struct {
	labels     labelMap
	value      float64
	metricType io_prometheus_client.MetricType
}

type mockShell struct {
	Output      []byte
	Err         error
	LastCommand string
}

func (t *mockShell) Execute(cmd string, args ...string) ([]byte, error) {
	t.LastCommand = cmd

	return t.Output, t.Err
}

func mockExporter(shell mockShell) exporter.Exporter {
	var cacheMap = make(map[string]twcli.CacheRecord)
	cli := twcli.TWCli{CacheDuration: 1, Cmd: "/fake/tw-cli", Cache: cacheMap, Shell: &shell}
	var controllerData []twcli.ControllerInfo
	controllerData = append(controllerData, twcli.ControllerInfo{
		Name: "/c4",
		Devices: []twcli.Device{
			{Name: "/c4/p0", Type: "SATA"},
		},
	})

	collector := exporter.Collector{ControllerData: controllerData, TWCli: cli}
	exporter := exporter.Exporter{Collector: &collector}

	return exporter
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

func TestCollectControllerDetails(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_all.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := mockShell{
		Output: output,
		Err:    nil,
	}

	e := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 1)
	result := e.Collector.CollectControllerDetails(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 1)

	expectedMetrics := labelMap{"available_memory": "234881024", "bios_version": "BE9X 4.08.00.004", "controller": "/c4", "firmware_version": "FE9X 4.10.00.027", "model": "9650SE-4LPML", "serial_number": "L1234568912345"}

	for metric := range ch {
		data := readMetric(metric)

		assert.Equal(t, 1.0, data.value)
		assert.Equal(t, io_prometheus_client.MetricType_GAUGE, data.metricType)
		assert.Equal(t, expectedMetrics, data.labels)
	}
}

func TestCollectUnitStatusOK(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_unitstatus_ok.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := mockShell{
		Output: output,
		Err:    nil,
	}

	e := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 1)
	result := e.Collector.CollectUnitStatus(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 1)

	expectedMetrics := labelMap{"controller": "/c4", "state": "OK", "type": "RAID-5", "unit": "u0"}

	for metric := range ch {
		data := readMetric(metric)

		assert.Equal(t, 1.0, data.value)
		assert.Equal(t, io_prometheus_client.MetricType_GAUGE, data.metricType)
		assert.Equal(t, expectedMetrics, data.labels)
	}
}

func TestCollectUnitStatusRebuilding(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_unitstatus_rebuilding.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := mockShell{
		Output: output,
		Err:    nil,
	}

	e := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 2)
	result := e.Collector.CollectUnitStatus(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 2)

	expectedLabels := map[int]labelMap{
		4: {"controller": "/c4", "state": "REBUILDING", "type": "RAID-5", "unit": "u0"},
		3: {"controller": "/c4", "state": "REBUILDING", "unit": "u0"},
	}

	expectedValues := map[int]float64{
		4: 0.0,
		3: 35.0,
	}

	for metric := range ch {
		data := readMetric(metric)
		assert.Equal(t, expectedValues[len(data.labels)], data.value)
		assert.Equal(t, io_prometheus_client.MetricType_GAUGE, data.metricType)
		assert.Equal(t, expectedLabels[len(data.labels)], data.labels)
	}
}

func TestCollectUnitStatusVerifying(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_unitstatus_verifying.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := mockShell{
		Output: output,
		Err:    nil,
	}

	e := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 2)
	result := e.Collector.CollectUnitStatus(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 2)

	expectedLabels := map[int]labelMap{
		4: {"controller": "/c4", "state": "VERIFYING", "type": "RAID-5", "unit": "u0"},
		3: {"controller": "/c4", "state": "VERIFYING", "unit": "u0"},
	}

	expectedValues := map[int]float64{
		4: 1.0,
		3: 21.0,
	}

	for metric := range ch {
		data := readMetric(metric)
		assert.Equal(t, expectedValues[len(data.labels)], data.value)
		assert.Equal(t, io_prometheus_client.MetricType_GAUGE, data.metricType)
		assert.Equal(t, expectedLabels[len(data.labels)], data.labels)
	}
}

func TestCollectDriveStatusOK(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_drivestatus_ok.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := mockShell{
		Output: output,
		Err:    nil,
	}

	e := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 4)
	result := e.Collector.CollectDriveStatus(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 4)

	expectedMetrics := map[string]labelMap{
		"0": {"status": "OK", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "0", "model": "ST4000VN006-3CW104"},
		"1": {"status": "OK", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "1", "model": "ST4000VN006-3CW104"},
		"2": {"status": "OK", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "2", "model": "TOSHIBA HDWG440"},
		"3": {"status": "OK", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "3", "model": "ST4000VN006-3CW104"},
	}

	for metric := range ch {
		data := readMetric(metric)
		assert.Equal(t, expectedMetrics[data.labels["phy"]], data.labels)
		assert.Equal(t, 1.0, data.value)
	}
}

func TestCollectDriveStatusDEGRADED(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_drivestatus_degraded.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := mockShell{
		Output: output,
		Err:    nil,
	}

	e := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 4)
	result := e.Collector.CollectDriveStatus(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 4)

	expectedMetrics := map[string]labelMap{
		"0": {"status": "OK", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "0", "model": "ST4000VN006-3CW104"},
		"1": {"status": "DEGRADED", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "1", "model": "ST4000VN006-3CW104"},
		"2": {"status": "OK", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "2", "model": "TOSHIBA HDWG440"},
		"3": {"status": "OK", "unit": "u0", "size": "3991227208827", "type": "SATA", "phy": "3", "model": "ST4000VN006-3CW104"},
	}

	for metric := range ch {
		data := readMetric(metric)
		assert.Equal(t, expectedMetrics[data.labels["phy"]], data.labels)

		if data.labels["status"] != "OK" {
			assert.Equal(t, 0.0, data.value)
		}
	}
}

func TestCollectDriveSmartData(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_drive_all_c4_p0.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := mockShell{
		Output: output,
		Err:    nil,
	}

	e := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 3)
	result := e.Collector.CollectDriveSmartData(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 3)

	expectedMetrics := []metricResult{
		{
			labels: labelMap{
				"status":        "OK",
				"model":         "ST4000VN006-3CW104",
				"serial":        "AA12345",
				"spindle_speed": "5400",
				"unit":          "u0",
			},
			value:      0,
			metricType: io_prometheus_client.MetricType_GAUGE,
		},
		{
			labels: labelMap{
				"status":        "OK",
				"model":         "ST4000VN006-3CW104",
				"serial":        "AA12345",
				"spindle_speed": "5400",
				"unit":          "u0",
			},
			value:      2355,
			metricType: io_prometheus_client.MetricType_COUNTER,
		},
		{
			labels: labelMap{
				"status":        "OK",
				"model":         "ST4000VN006-3CW104",
				"serial":        "AA12345",
				"spindle_speed": "5400",
				"unit":          "u0",
			},
			value:      31,
			metricType: io_prometheus_client.MetricType_GAUGE,
		},
	}

	i := 0
	for metric := range ch {
		data := readMetric(metric)
		assert.Equal(t, expectedMetrics[i].labels, data.labels)
		assert.Equal(t, expectedMetrics[i].value, data.value)
		assert.Equal(t, expectedMetrics[i].metricType, data.metricType)

		i++
	}
}

type mockCollector struct {
	ctrlOK, unitOK, driveOK, smartOK bool
}

func (m *mockCollector) CollectControllerDetails(ch chan<- prometheus.Metric) bool {
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

func TestExporterCollectOK(t *testing.T) {
	ch := make(chan prometheus.Metric, 2)
	e := &exporter.Exporter{
		Collector: &mockCollector{true, true, true, true},
	}
	e.Collect(ch)
	close(ch)

	assert.Len(t, ch, 2)
	for metric := range ch {
		desc := metric.Desc().String()
		if strings.Contains(desc, "tw_cli_scrape_collector_success") {
			data := readMetric(metric)
			assert.Equal(t, 1.0, data.value)
		}
	}
}

func TestExporterCollectFail(t *testing.T) {
	ch := make(chan prometheus.Metric, 2)
	e := &exporter.Exporter{
		Collector: &mockCollector{false, true, true, true},
	}
	e.Collect(ch)
	close(ch)

	assert.Len(t, ch, 2)
	for metric := range ch {
		desc := metric.Desc().String()
		if strings.Contains(desc, "tw_cli_scrape_collector_success") {
			data := readMetric(metric)
			assert.Equal(t, 0.0, data.value)
		}
	}
}
