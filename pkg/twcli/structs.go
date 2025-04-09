package twcli

import (
	"time"

	"github.com/pritpal-sabharwal/prometheus-twcli-exporter/pkg/shell"
)

type TWCli struct {
	Shell         shell.Shell
	Cmd           string
	Cache         map[string]CacheRecord
	CacheDuration int
}

type DriveLabels struct {
	Status string
	Unit   string
	Size   string
	Type   string
	Phy    string
	Model  string
}

type CacheRecord struct {
	ExpiresAt time.Time
	Data      []byte
}
