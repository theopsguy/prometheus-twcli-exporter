package twcli

import "strconv"

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
