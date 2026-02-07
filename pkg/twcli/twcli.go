package twcli

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/theopsguy/prometheus-twcli-exporter/pkg/shell"
)

type Client interface {
	GetControllers() ([]string, error)
	GetControllerInfo(controller string) (ControllerInfo, error)
	GetUnitStatus(controller string) (UnitStatus, error)
	GetDriveStatus(controller string) ([]DriveInfo, error)
	GetSATASmartData(controller, device string) (SATASmartData, error)
}

type TWCli struct {
	Shell         shell.Shell
	Cmd           string
	Cache         map[string]CacheRecord
	CacheDuration int
}

type ControllerInventory struct {
	Name    string
	Devices []Device
}

type Device struct {
	Name string
	Type string
}

type ControllerInfo struct {
	Controller      string
	Model           string
	AvailableMemory string
	FirmwareVersion string
	BiosVersion     string
	SerialNumber    string
}

type UnitStatus struct {
	Unit            string
	Type            string
	State           string
	PercentComplete float64
}

type DriveInfo struct {
	Status string
	Unit   string
	Size   string
	Type   string
	Phy    string
	Model  string
}

type SATASmartData struct {
	Controller         string
	Device             string
	Status             string
	Model              string
	Serial             string
	Unit               string
	ReallocatedSectors string
	PowerOnHours       string
	Temperature        string
	SpindleSpeed       string
}

type CacheRecord struct {
	ExpiresAt time.Time
	Data      []byte
}

func New(cacheDuration int, executable string, shell shell.Shell) *TWCli {
	cacheMap := make(map[string]CacheRecord)

	return &TWCli{
		Shell:         shell,
		Cmd:           executable,
		Cache:         cacheMap,
		CacheDuration: cacheDuration,
	}
}

func (twcli *TWCli) RunCommand(args ...string) ([]byte, error) {

	cacheKey := strings.Join(args, ":")
	value, ok := twcli.Cache[cacheKey]
	if ok && value.ExpiresAt.After(time.Now()) {
		return value.Data, nil
	}

	output, err := twcli.Shell.Execute(twcli.Cmd, args...)

	if err != nil {
		slog.Error("Error running command", "error", err)
		return output, err
	}

	cacheExpiry := time.Now().Add(time.Duration(twcli.CacheDuration) * time.Second)
	twcli.Cache[cacheKey] = CacheRecord{ExpiresAt: cacheExpiry, Data: output}

	return output, nil
}

func (twcli *TWCli) GetControllers() ([]string, error) {
	var controllers []string
	output, err := twcli.RunCommand("show")
	if err != nil {
		return controllers, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "c") {
			c := strings.Split(line, " ")
			controllers = append(controllers, "/"+c[0])
		}
	}

	return controllers, nil
}

func (twcli *TWCli) GetDevices(controller string) ([]Device, error) {
	var devices []Device
	re := regexp.MustCompile(`^\s*phy\d+\s+-\s+(\S+)\s+(\S+)`)

	output, err := twcli.RunCommand(controller, "show", "phy")
	if err != nil {
		return devices, err
	}

	for line := range strings.SplitSeq(string(output), "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			devices = append(devices, Device{
				Type: matches[1],
				Name: matches[2],
			})
		}
	}

	return devices, nil
}

func (twcli *TWCli) GetControllerInfo(controller string) (ControllerInfo, error) {
	info := ControllerInfo{
		Controller: controller,
	}

	output, err := twcli.RunCommand(controller, "show", "all")
	if err != nil {
		return info, err
	}

	fields := []struct {
		outputName   string
		target       *string
		needsConvert bool
	}{
		{"Model", &info.Model, false},
		{"Available Memory", &info.AvailableMemory, true},
		{"Firmware Version", &info.FirmwareVersion, false},
		{"Bios Version", &info.BiosVersion, false},
		{"Serial Number", &info.SerialNumber, false},
	}

	for _, field := range fields {
		pattern := fmt.Sprintf(`%s\s*%s\s*=\s*(.*)`, controller, field.outputName)
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(string(output))

		if len(matches) != 2 {
			slog.Warn("Field not found", "field", field.outputName, "controller", controller)
			continue
		}

		value := matches[1]
		if field.needsConvert {
			number, unit := parseAvailableMemory(value)
			value, err = convertToBytes(number, unit)
			if err != nil {
				return info, err
			}
		}
		*field.target = value
	}

	return info, nil
}

func (twcli *TWCli) GetUnitStatus(controller string) (UnitStatus, error) {
	status := UnitStatus{}

	output, err := twcli.RunCommand(controller, "show", "unitstatus")
	if err != nil {
		return status, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "u") {
			fields := strings.Fields(line)

			status.Unit = fields[0]
			status.Type = fields[1]
			status.State = fields[2]
			rebuildPercent := fields[3]
			verifyingPercent := fields[4]

			if status.State == "REBUILDING" {
				rebuildValue := strings.TrimSuffix(rebuildPercent, "%")
				status.PercentComplete, _ = strconv.ParseFloat(rebuildValue, 64)
			}

			if status.State == "VERIFYING" {
				verifyingValue := strings.TrimSuffix(verifyingPercent, "%")
				status.PercentComplete, _ = strconv.ParseFloat(verifyingValue, 64)
			}

		}
	}

	return status, nil
}

func (twcli *TWCli) GetDriveStatus(controller string) ([]DriveInfo, error) {
	var drives []DriveInfo

	output, err := twcli.RunCommand(controller, "show", "drivestatus")
	if err != nil {
		return drives, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "p") {
			fields := strings.Fields(line)
			lineLength := len(fields)

			driveStatus := fields[1]
			unit := fields[2]
			driveSize := fields[3]
			driveSizeUnit := fields[4]
			driveSizeBytes, _ := convertToBytes(driveSize, driveSizeUnit)

			driveType := fields[5]
			drivePhy := fields[6]
			driveModel := fields[8]

			if lineLength > 9 {
				driveModel = fmt.Sprintf("%s %s", fields[8], fields[9])
			}

			drive := DriveInfo{
				Status: driveStatus,
				Unit:   unit,
				Size:   driveSizeBytes,
				Type:   driveType,
				Phy:    drivePhy,
				Model:  driveModel,
			}
			drives = append(drives, drive)
		}
	}

	return drives, nil
}

func (twcli *TWCli) GetSATASmartData(controller string, device string) (SATASmartData, error) {
	data := SATASmartData{
		Controller: controller,
		Device:     device,
	}

	output, err := twcli.RunCommand(device, "show", "all")
	if err != nil {
		return data, err
	}

	fieldMap := map[string]*string{
		"Status":              &data.Status,
		"Model":               &data.Model,
		"Serial":              &data.Serial,
		"Belongs to Unit":     &data.Unit,
		"Reallocated Sectors": &data.ReallocatedSectors,
		"Power On Hours":      &data.PowerOnHours,
		"Temperature":         &data.Temperature,
		"Spindle Speed":       &data.SpindleSpeed,
	}

	for field, ptr := range fieldMap {
		pattern := fmt.Sprintf(`(?i)%s\s+%s\s*=\s*(.*)`, regexp.QuoteMeta(device), regexp.QuoteMeta(field))
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(string(output))

		if len(matches) != 2 {
			slog.Warn("Field not found", "field", field, "device", device)
			continue
		}
		value := matches[1]

		if field == "Temperature" || field == "Spindle Speed" {
			value = strings.Fields(value)[0]
		}

		*ptr = value
	}

	return data, nil
}
