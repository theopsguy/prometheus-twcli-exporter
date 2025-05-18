package exporter

import (
	"log"
	"strconv"
)

func parseFloat(value string, fieldName string) (float64, bool) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("Warning: could not parse '%s' for field '%s': %v", value, fieldName, err)
		parseErrors.WithLabelValues(fieldName).Inc()
		return 0.0, false
	}
	return f, true
}
