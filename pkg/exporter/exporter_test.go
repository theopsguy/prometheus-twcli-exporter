package exporter_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	testutil "github.com/pritpal-sabharwal/prometheus-twcli-exporter/internal/testutil"
	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/config"
	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/exporter"
	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/twcli"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

type labelMap map[string]string

type MetricResult struct {
	labels     labelMap
	value      float64
	metricType dto.MetricType
}

type MockShell struct {
	Output      []byte
	Err         error
	LastCommand string
}

type mockCollector struct {
	ctrlOK, unitOK, driveOK bool
}

func (m mockCollector) CollectControllerDetails(ch chan<- prometheus.Metric) bool { return m.ctrlOK }
func (m mockCollector) CollectUnitStatus(ch chan<- prometheus.Metric) bool        { return m.unitOK }
func (m mockCollector) CollectDriveStatus(ch chan<- prometheus.Metric) bool       { return m.driveOK }

func (t *MockShell) Execute(cmd string, args ...string) ([]byte, error) {
	t.LastCommand = cmd

	return t.Output, t.Err
}

func mockExporter(shell MockShell) exporter.Exporter {
	var cacheMap = make(map[string]twcli.CacheRecord)
	cli := twcli.TWCli{CacheDuration: 1, Cmd: "/fake/tw-cli", Cache: cacheMap, Shell: &shell}
	exporter := exporter.Exporter{Controllers: []string{"/c4"}, TWCli: cli}

	return exporter
}

func readMetric(m prometheus.Metric) MetricResult {
	pb := &dto.Metric{}
	m.Write(pb)
	labels := make(labelMap, len(pb.Label))
	for _, v := range pb.Label {
		labels[v.GetName()] = v.GetValue()
	}

	if pb.Gauge != nil {
		return MetricResult{labels: labels, value: pb.GetGauge().GetValue(), metricType: dto.MetricType_GAUGE}
	}
	if pb.Counter != nil {
		return MetricResult{labels: labels, value: pb.GetCounter().GetValue(), metricType: dto.MetricType_COUNTER}
	}
	if pb.Summary != nil {
		return MetricResult{labels: labels, value: pb.GetSummary().GetSampleSum(), metricType: dto.MetricType_SUMMARY}
	}
	if pb.Untyped != nil {
		return MetricResult{labels: labels, value: pb.GetUntyped().GetValue(), metricType: dto.MetricType_UNTYPED}
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
	assert.Contains(t, stderr.String(), "Error running command: fork/exec /usr/sbin/tw-cli: no such file or directory")
}

func TestCollectControllerDetails(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_all.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: output,
		Err:    nil,
	}

	exporter := mockExporter(mshell)

	ch := make(chan prometheus.Metric, 1)
	result := exporter.CollectControllerDetails(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 1)

	expectedMetrics := labelMap{"available_memory": "234881024", "bios_version": "BE9X 4.08.00.004", "controller": "/c4", "firmware_version": "FE9X 4.10.00.027", "model": "9650SE-4LPML", "serial_number": "L1234568912345"}

	for metric := range ch {
		data := readMetric(metric)

		assert.Equal(t, data.value, 1.0)
		assert.Equal(t, data.metricType, io_prometheus_client.MetricType_GAUGE)
		assert.Equal(t, data.labels, expectedMetrics)
	}
}

func TestCollectUnitStatusOK(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_unitstatus_ok.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: output,
		Err:    nil,
	}

	exporter := mockExporter(mshell)

	ch := make(chan prometheus.Metric, 1)
	result := exporter.CollectUnitStatus(ch)
	close(ch)

	assert.True(t, result)
	assert.Len(t, ch, 1)

	expectedMetrics := labelMap{"controller": "/c4", "state": "OK", "type": "RAID-5", "unit": "u0"}

	for metric := range ch {
		data := readMetric(metric)

		assert.Equal(t, data.value, 1.0)
		assert.Equal(t, data.metricType, io_prometheus_client.MetricType_GAUGE)
		assert.Equal(t, data.labels, expectedMetrics)
	}
}

func TestCollectUnitStatusRebuilding(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_unitstatus_rebuilding.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: output,
		Err:    nil,
	}

	exporter := mockExporter(mshell)

	ch := make(chan prometheus.Metric, 2)
	result := exporter.CollectUnitStatus(ch)
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
		assert.Equal(t, data.value, expectedValues[len(data.labels)])
		assert.Equal(t, data.metricType, io_prometheus_client.MetricType_GAUGE)
		assert.Equal(t, data.labels, expectedLabels[len(data.labels)])
	}
}

func TestCollectUnitStatusVerifying(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_unitstatus_verifying.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: output,
		Err:    nil,
	}

	exporter := mockExporter(mshell)

	ch := make(chan prometheus.Metric, 2)
	result := exporter.CollectUnitStatus(ch)
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
		assert.Equal(t, data.value, expectedValues[len(data.labels)])
		assert.Equal(t, data.metricType, io_prometheus_client.MetricType_GAUGE)
		assert.Equal(t, data.labels, expectedLabels[len(data.labels)])
	}
}

func TestCollectDriveStatusOK(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_drivestatus_ok.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: output,
		Err:    nil,
	}

	exporter := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 4)
	result := exporter.CollectDriveStatus(ch)
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
		assert.Equal(t, data.labels, expectedMetrics[data.labels["phy"]])
		assert.Equal(t, data.value, 1.0)
	}
}

func TestCollectDriveStatusDEGRADED(t *testing.T) {
	output, err := testutil.ReadTestOutputData("testdata/show_drivestatus_degraded.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: output,
		Err:    nil,
	}

	exporter := mockExporter(mshell)
	ch := make(chan prometheus.Metric, 4)
	result := exporter.CollectDriveStatus(ch)
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
		assert.Equal(t, data.labels, expectedMetrics[data.labels["phy"]])

		if data.labels["status"] != "OK" {
			assert.Equal(t, data.value, 0.0)
		}
	}
}

func TestExporterCollectOK(t *testing.T) {
	ch := make(chan prometheus.Metric, 2)
	e := exporter.Exporter{
		Collector: mockCollector{
			ctrlOK:  true,
			unitOK:  true,
			driveOK: true,
		},
	}
	e.Collect(ch)
	close(ch)

	assert.Len(t, ch, 2)
	for metric := range ch {
		desc := metric.Desc().String()
		if strings.Contains(desc, "tw_cli_scrape_collector_success") {
			data := readMetric(metric)
			assert.Equal(t, data.value, 1.0)
		}
	}
}

func TestExporterCollectFail(t *testing.T) {
	ch := make(chan prometheus.Metric, 2)
	e := exporter.Exporter{
		Collector: mockCollector{
			ctrlOK:  false,
			unitOK:  true,
			driveOK: true,
		},
	}
	e.Collect(ch)
	close(ch)

	assert.Len(t, ch, 2)
	for metric := range ch {
		desc := metric.Desc().String()
		if strings.Contains(desc, "tw_cli_scrape_collector_success") {
			data := readMetric(metric)
			assert.Equal(t, data.value, 0.0)
		}
	}
}
