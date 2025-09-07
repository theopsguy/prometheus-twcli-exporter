package twcli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/theopsguy/prometheus-twcli-exporter/internal/testutil"
	"github.com/theopsguy/prometheus-twcli-exporter/pkg/twcli"
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
	assert.Equal(t, []string{"/c4"}, output)
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
	assert.Equal(t, expectedOutput, output)
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
	assert.Equal(t, []string{"/c4", "9650SE-4LPML", "234881024", "FE9X 4.10.00.027", "BE9X 4.08.00.004", "L1234568912345"}, output)
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
	assert.Equal(t, "u0", unit)
	assert.Equal(t, "RAID-5", unitType)
	assert.Equal(t, "OK", unitStatus)
	assert.Equal(t, 0, percentComplete)
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
	assert.Equal(t, "u0", unit)
	assert.Equal(t, "RAID-5", unitType)
	assert.Equal(t, "REBUILDING", unitStatus)
	assert.Equal(t, 35, percentComplete)
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
	assert.Equal(t, "u0", unit)
	assert.Equal(t, "RAID-5", unitType)
	assert.Equal(t, "VERIFYING", unitStatus)
	assert.Equal(t, 21, percentComplete)
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
	assert.Equal(t, expectedOutput, drives)
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
	assert.Equal(t, expectedOutput, drives)
}

type deviceTestData struct {
	Device         string
	TestDataFile   string
	ExpectedOutput *twcli.SATASmartData
}

func TestGetSATASmartData(t *testing.T) {
	devices := []deviceTestData{
		{
			Device:         "/c4/p0",
			TestDataFile:   "testdata/show_drive_all_c4_p0.txt",
			ExpectedOutput: &twcli.SATASmartData{Controller: "/c4", Device: "/c4/p0", Status: "OK", Model: "ST4000VN006-3CW104", Serial: "AA12345", Unit: "u0", ReallocatedSectors: "0", PowerOnHours: "2355", Temperature: "31", SpindleSpeed: "5400"},
		},
		{
			Device:         "/c4/p1",
			TestDataFile:   "testdata/show_drive_all_c4_p1.txt",
			ExpectedOutput: &twcli.SATASmartData{Controller: "/c4", Device: "/c4/p1", Status: "OK", Model: "ST4000VN006-3CW104", Serial: "AB12345", Unit: "u0", ReallocatedSectors: "0", PowerOnHours: "2453", Temperature: "31", SpindleSpeed: "5400"},
		},
		{
			Device:         "/c4/p2",
			TestDataFile:   "testdata/show_drive_all_c4_p2.txt",
			ExpectedOutput: &twcli.SATASmartData{Controller: "/c4", Device: "/c4/p2", Status: "OK", Model: "TOSHIBA HDWG440", Serial: "AC12345", Unit: "u0", ReallocatedSectors: "0", PowerOnHours: "20120", Temperature: "27", SpindleSpeed: "7200"},
		},
		{
			Device:         "/c4/p3",
			TestDataFile:   "testdata/show_drive_all_c4_p3.txt",
			ExpectedOutput: &twcli.SATASmartData{Controller: "/c4", Device: "/c4/p3", Status: "OK", Model: "ST4000VN006-3CW104", Serial: "AD12345", Unit: "u0", ReallocatedSectors: "0", PowerOnHours: "2349", Temperature: "31", SpindleSpeed: "5400"},
		},
	}

	for _, d := range devices {
		testdata, err := testutil.ReadTestOutputData(d.TestDataFile)
		if err != nil {
			t.Fatalf("Error reading test data: %s", err)
		}

		mshell := MockShell{
			Output: testdata,
			Err:    nil,
		}

		twcli := mockTWCli(mshell)
		labels, _ := twcli.GetSATASmartData("/c4", d.Device)
		assert.Equal(t, d.ExpectedOutput, labels)
	}
}
