package twcli

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/shell"
)

type TWCli struct {
	Shell         shell.Shell
	Cmd           string
	Cache         map[string]CacheRecord
	CacheDuration int
}

type ControllerInfo struct {
	Name    string
	Devices []Device
}

type Device struct {
	Name string
	Type string
}

type DriveLabels struct {
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
		log.Printf("Error running command: %s\n", err)
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

func (twcli *TWCli) GetControllerInfo(controller string) ([]string, error) {
	labels := []string{controller}

	output, err := twcli.RunCommand(controller, "show", "all")
	if err != nil {
		return labels, err
	}

	fields := []string{"Model", "Available Memory", "Firmware Version", "Bios Version", "Serial Number"}

	for _, field := range fields {
		pattern := fmt.Sprintf(`%s\s*%s\s*=\s*(.*)`, controller, field)
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(string(output))

		if len(matches) != 2 {
			continue
		}

		value := matches[1]
		if field == "Available Memory" {
			number, unit := parseAvailableMemory(value)
			value, err = convertToBytes(number, unit)
			if err != nil {
				return labels, err
			}
		}

		labels = append(labels, value)
	}

	return labels, nil
}

func (twcli *TWCli) GetUnitStatus(controller string) (string, string, string, int, error) {
	var unit, unitType, unitStatus string
	var percentComplete int

	output, err := twcli.RunCommand(controller, "show", "unitstatus")
	if err != nil {
		return unit, unitType, unitStatus, percentComplete, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "u") {
			unitDetails := strings.Fields(line)

			unit = unitDetails[0]
			unitType = unitDetails[1]
			unitStatus = unitDetails[2]
			rebuildPercent := unitDetails[3]
			verifyingPercent := unitDetails[4]

			if unitStatus == "REBUILDING" {
				rebuildValue := strings.TrimSuffix(rebuildPercent, "%")
				percentComplete, _ = strconv.Atoi(rebuildValue)
			}

			if unitStatus == "VERIFYING" {
				verifyingValue := strings.TrimSuffix(verifyingPercent, "%")
				percentComplete, _ = strconv.Atoi(verifyingValue)
			}

		}
	}

	return unit, unitType, unitStatus, percentComplete, nil
}

func (twcli *TWCli) GetDriveStatus(controller string) ([]DriveLabels, error) {
	var drives []DriveLabels

	output, err := twcli.RunCommand(controller, "show", "drivestatus")
	if err != nil {
		return drives, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "p") {
			driveDetails := strings.Fields(line)
			lineLength := len(driveDetails)

			driveStatus := driveDetails[1]
			unit := driveDetails[2]
			driveSize := driveDetails[3]
			driveSizeUnit := driveDetails[4]
			driveSizeBytes, _ := convertToBytes(driveSize, driveSizeUnit)

			driveType := driveDetails[5]
			drivePhy := driveDetails[6]
			driveModel := driveDetails[8]

			if lineLength > 9 {
				driveModel = fmt.Sprintf("%s %s", driveDetails[8], driveDetails[9])
			}

			labels := DriveLabels{
				Status: driveStatus,
				Unit:   unit,
				Size:   driveSizeBytes,
				Type:   driveType,
				Phy:    drivePhy,
				Model:  driveModel,
			}
			drives = append(drives, labels)
		}
	}

	return drives, nil
}

func (twcli *TWCli) GetSATASmartData(controller string, device string) (*SATASmartData, error) {
	data := &SATASmartData{
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
			log.Printf("Field '%s' not found for device '%s'", field, device)
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
