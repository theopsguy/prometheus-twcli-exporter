package shell

import (
	"os/exec"
)

type LocalShell struct{}

func (LocalShell) Execute(cmd string, args ...string) ([]byte, error) {
	wrapperCmd := exec.Command(cmd, args...)
	output, err := wrapperCmd.CombinedOutput()

	return output, err
}
