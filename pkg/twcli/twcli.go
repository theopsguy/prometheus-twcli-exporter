package twcli

import (
	"log"
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

// TODO: Maybe change this to be a little more generic
type parseOptions struct {
	HasUnits bool
	Units    string
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

func (twcli *TWCli) GetControllerInfo(controller string) ([]string, error) {
	var labels []string

	output, err := twcli.RunCommand(controller, "show", "all")
	if err != nil {
		return labels, err
	}

	var controllerFields = map[string]parseOptions{
		"Model":            {HasUnits: false, Units: ""},
		"Available Memory": {HasUnits: true, Units: "MB"},
		"Firmware Version": {HasUnits: false, Units: ""},
		"Bios Version":     {HasUnits: false, Units: ""},
		"Serial Number":    {HasUnits: false, Units: ""},
	}

	labels = append(labels, controller)
	for _, line := range strings.Split(string(output), "\n") {
		prefix := controller + " "
		if strings.HasPrefix(line, prefix) {
			splitLine := strings.Split(line, "=")
			key := strings.Trim(splitLine[0], " ")
			options, ok := controllerFields[strings.TrimPrefix(key, prefix)]
			if ok {
				value := strings.Split(line, "=")[1]

				if options.HasUnits {
					value = strings.TrimSuffix(value, options.Units)
				}

				labels = append(labels, strings.TrimSpace(value))
			}
		}
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
