package twcli_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

func readTestOutputData(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func TestGetControllers(t *testing.T) {
	tests := []struct {
		name           string
		testDataFile   string
		expectedOutput []string
	}{
		{
			name:           "Single Controller",
			testDataFile:   "testdata/show_single_controller.txt",
			expectedOutput: []string{"/c4"},
		},
		{
			name:           "Multiple Controllers",
			testDataFile:   "testdata/show_multiple_controllers.txt",
			expectedOutput: []string{"/c3", "/c4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testdata, err := readTestOutputData(tt.testDataFile)
			if err != nil {
				t.Fatalf("Error reading test data: %s", err)
			}
			mshell := MockShell{Output: testdata}

			twcli := mockTWCli(mshell)
			output, err := twcli.GetControllers()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestGetDevices(t *testing.T) {
	tests := []struct {
		name           string
		testDataFile   string
		controller     string
		expectedOutput []twcli.Device
	}{
		{
			name:         "OK",
			testDataFile: "testdata/show_phy.txt",
			controller:   "/c4",
			expectedOutput: []twcli.Device{
				{Name: "/c4/p0", Type: "SATA"},
				{Name: "/c4/p1", Type: "SATA"},
				{Name: "/c4/p2", Type: "SATA"},
				{Name: "/c4/p3", Type: "SATA"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testdata, err := readTestOutputData(tt.testDataFile)
			if err != nil {
				t.Fatalf("Error reading test data: %s", err)
			}
			mshell := MockShell{Output: testdata}

			twcli := mockTWCli(mshell)
			output, err := twcli.GetDevices(tt.controller)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestGetControllerInfo(t *testing.T) {
	tests := []struct {
		name           string
		testDataFile   string
		controller     string
		expectedOutput twcli.ControllerInfo
	}{
		{
			name:         "OK",
			testDataFile: "testdata/show_all.txt",
			controller:   "/c4",
			expectedOutput: twcli.ControllerInfo{
				Controller:      "/c4",
				AvailableMemory: "234881024",
				BiosVersion:     "BE9X 4.08.00.004",
				FirmwareVersion: "FE9X 4.10.00.027",
				Model:           "9650SE-4LPML",
				SerialNumber:    "L1234568912345",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testdata, err := readTestOutputData(tt.testDataFile)
			if err != nil {
				t.Fatalf("Error reading test data: %s", err)
			}
			mshell := MockShell{Output: testdata}

			mock := mockTWCli(mshell)
			output, err := mock.GetControllerInfo(tt.controller)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestGetUnitStatus(t *testing.T) {
	tests := []struct {
		name           string
		testDataFile   string
		controller     string
		expectedOutput twcli.UnitStatus
	}{
		{
			name:         "OK",
			testDataFile: "testdata/show_unitstatus_ok.txt",
			controller:   "/c4",
			expectedOutput: twcli.UnitStatus{
				Unit:            "u0",
				Type:            "RAID-5",
				State:           "OK",
				PercentComplete: 0.0,
			},
		},
		{
			name:         "REBUILDING",
			testDataFile: "testdata/show_unitstatus_rebuilding.txt",
			controller:   "/c4",
			expectedOutput: twcli.UnitStatus{
				Unit:            "u0",
				Type:            "RAID-5",
				State:           "REBUILDING",
				PercentComplete: 35.0,
			},
		},
		{
			name:         "VERIFYING",
			testDataFile: "testdata/show_unitstatus_verifying.txt",
			controller:   "/c4",
			expectedOutput: twcli.UnitStatus{
				Unit:            "u0",
				Type:            "RAID-5",
				State:           "VERIFYING",
				PercentComplete: 21.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testdata, err := readTestOutputData(tt.testDataFile)
			if err != nil {
				t.Fatalf("Error reading test data: %s", err)
			}
			mshell := MockShell{Output: testdata}

			twcli := mockTWCli(mshell)
			status, err := twcli.GetUnitStatus(tt.controller)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, status)
		})
	}
}

func TestGetDriveStatus(t *testing.T) {
	tests := []struct {
		name           string
		testDataFile   string
		controller     string
		expectedOutput []twcli.DriveInfo
	}{
		{
			name:         "OK",
			testDataFile: "testdata/show_drivestatus_ok.txt",
			controller:   "/c4",
			expectedOutput: []twcli.DriveInfo{
				{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "0", Model: "ST4000VN006-3CW104"},
				{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "1", Model: "ST4000VN006-3CW104"},
				{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "2", Model: "TOSHIBA HDWG440"},
				{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "3", Model: "ST4000VN006-3CW104"},
			},
		},
		{
			name:         "DEGRADED",
			testDataFile: "testdata/show_drivestatus_degraded.txt",
			controller:   "/c4",
			expectedOutput: []twcli.DriveInfo{
				{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "0", Model: "ST4000VN006-3CW104"},
				{Status: "DEGRADED", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "1", Model: "ST4000VN006-3CW104"},
				{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "2", Model: "TOSHIBA HDWG440"},
				{Status: "OK", Unit: "u0", Size: "3991227208827", Type: "SATA", Phy: "3", Model: "ST4000VN006-3CW104"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testdata, err := readTestOutputData(tt.testDataFile)
			if err != nil {
				t.Fatalf("Error reading test data: %s", err)
			}
			mshell := MockShell{Output: testdata}

			twcli := mockTWCli(mshell)
			drives, err := twcli.GetDriveStatus(tt.controller)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, drives)
		})
	}
}

func TestGetSATASmartData(t *testing.T) {
	tests := []struct {
		name           string
		testDataFile   string
		controller     string
		device         string
		expectedOutput twcli.SATASmartData
	}{
		{
			name:         "OK",
			testDataFile: "testdata/show_drive_all_c4_p0.txt",
			controller:   "/c4",
			device:       "/c4/p0",
			expectedOutput: twcli.SATASmartData{
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
		{
			name:         "DEGRADED",
			testDataFile: "testdata/show_drive_all_c4_p1.txt",
			controller:   "/c4",
			device:       "/c4/p1",
			expectedOutput: twcli.SATASmartData{
				Controller:         "/c4",
				Device:             "/c4/p1",
				Status:             "DEGRADED",
				Model:              "ST4000VN006-3CW104",
				Serial:             "AB12345",
				Unit:               "u0",
				ReallocatedSectors: "0",
				PowerOnHours:       "2453",
				Temperature:        "31",
				SpindleSpeed:       "5400",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testdata, err := readTestOutputData(tt.testDataFile)
			if err != nil {
				t.Fatalf("Error reading test data: %s", err)
			}
			mshell := MockShell{Output: testdata}

			twcli := mockTWCli(mshell)
			labels, err := twcli.GetSATASmartData(tt.controller, tt.device)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, labels)
		})
	}
}
