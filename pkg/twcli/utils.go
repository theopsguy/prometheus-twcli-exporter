package twcli

import (
	"regexp"
	"strconv"
)

func convertToBytes(size string, unit string) (string, error) {
	var convertedSize float64
	sizeInt, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return "", err
	}

	switch unit {
	case "TB":
		convertedSize = sizeInt * 1024 * 1024 * 1024 * 1024
	case "GB":
		convertedSize = sizeInt * 1024 * 1024 * 1024
	case "MB":
		convertedSize = sizeInt * 1024 * 1024
	}

	return strconv.FormatFloat(convertedSize, 'f', 0, 64), nil
}

func parseAvailableMemory(input string) (string, string) {
	var number, unit string

	pattern := `^(\d+)([a-zA-Z]+)$`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)

	if len(matches) == 3 {
		number = matches[1]
		unit = matches[2]
	}

	return number, unit
}
