package twcli_test

import (
	"testing"

	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/internal/testutil"
	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/twcli"
	"github.com/stretchr/testify/assert"
)

type MockShell struct {
	Output      []byte
	Err         error
	LastCommand string
}

func (t *MockShell) Execute(cmd string, args ...string) ([]byte, error) {
	t.LastCommand = cmd

	return t.Output, t.Err
}

func mockTWCli(shell MockShell) twcli.TWCli {
	cacheMap := make(map[string]twcli.CacheRecord)
	twcli := twcli.TWCli{CacheDuration: 1, Cmd: "/fake/tw-cli", Cache: cacheMap, Shell: &shell}

	return twcli
}

func TestGetControllers(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	twcli := mockTWCli(mshell)
	output, err := twcli.GetControllers()
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, output, []string{"/c4"})
}

func TestGetDevices(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show_phy.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	expectedOutput := []twcli.Device{
		{Name: "/c4/p0", Type: "SATA"},
		{Name: "/c4/p1", Type: "SATA"},
		{Name: "/c4/p2", Type: "SATA"},
		{Name: "/c4/p3", Type: "SATA"},
	}

	twcli := mockTWCli(mshell)
	output, err := twcli.GetDevices("/c4")
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, output, expectedOutput)
}

func TestGetControllerInfo(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show_all.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	twcli := mockTWCli(mshell)
	output, err := twcli.GetControllerInfo("/c4")
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, output, []string{"/c4", "9650SE-4LPML", "234881024", "FE9X 4.10.00.027", "BE9X 4.08.00.004", "L1234568912345"})
}

func TestGetUnitStatusOK(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show_unitstatus_ok.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	twcli := mockTWCli(mshell)
	unit, unitType, unitStatus, percentComplete, err := twcli.GetUnitStatus("/c4")
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, unit, "u0")
	assert.Equal(t, unitType, "RAID-5")
	assert.Equal(t, unitStatus, "OK")
	assert.Equal(t, percentComplete, 0)
}

func TestGetUnitStatusREBUILDING(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show_unitstatus_rebuilding.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	twcli := mockTWCli(mshell)
	unit, unitType, unitStatus, percentComplete, err := twcli.GetUnitStatus("/c4")
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, unit, "u0")
	assert.Equal(t, unitType, "RAID-5")
	assert.Equal(t, unitStatus, "REBUILDING")
	assert.Equal(t, percentComplete, 35)
}

func TestGetUnitStatusVERIFYING(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show_unitstatus_verifying.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	twcli := mockTWCli(mshell)
	unit, unitType, unitStatus, percentComplete, err := twcli.GetUnitStatus("/c4")
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, unit, "u0")
	assert.Equal(t, unitType, "RAID-5")
	assert.Equal(t, unitStatus, "VERIFYING")
	assert.Equal(t, percentComplete, 21)
}

func TestGetDriveStatusOK(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show_drivestatus_ok.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	expectedOutput := []twcli.DriveLabels{
		{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "0", Model: "ST4000VN006-3CW104"},
		{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "1", Model: "ST4000VN006-3CW104"},
		{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "2", Model: "TOSHIBA HDWG440"},
		{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "3", Model: "ST4000VN006-3CW104"},
	}

	twcli := mockTWCli(mshell)
	drives, err := twcli.GetDriveStatus("/c4")
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, drives, expectedOutput)
}

func TestGetDriveStatusDEGRADED(t *testing.T) {
	testdata, err := testutil.ReadTestOutputData("testdata/show_drivestatus_degraded.txt")
	if err != nil {
		t.Fatalf("Error reading test data: %s", err)
	}
	mshell := MockShell{
		Output: testdata,
		Err:    nil,
	}

	expectedOutput := []twcli.DriveLabels{
		{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "0", Model: "ST4000VN006-3CW104"},
		{Status: "DEGRADED", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "1", Model: "ST4000VN006-3CW104"},
		{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "2", Model: "TOSHIBA HDWG440"},
		{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "3", Model: "ST4000VN006-3CW104"},
	}

	twcli := mockTWCli(mshell)
	drives, err := twcli.GetDriveStatus("/c4")
	assert.Nil(t, err, "unexpected error: %v", err)
	assert.Equal(t, drives, expectedOutput)
}
