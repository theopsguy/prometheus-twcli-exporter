package testutil

import (
	"os"
)

func ReadTestOutputData(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return data, nil
}
