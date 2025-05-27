package exporter

import (
	"strconv"

	"log/slog"
)

func parseFloat(value string, fieldName string) (float64, bool) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		slog.Error("Unable to parse float", "field", fieldName, "value", value, "error", err)
		parseErrors.WithLabelValues(fieldName).Inc()
		return 0.0, false
	}
	return f, true
}
