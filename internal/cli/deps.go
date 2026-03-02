package cli

import (
	"fmt"
	"os/exec"
)

// checkDep verifies a CLI tool is on PATH. Returns an error if not found.
func checkDep(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s CLI not found on PATH", name)
	}
	return nil
}

// checkBd verifies bd is available. Required for all commands.
func checkBd() error {
	return checkDep("bd")
}

// checkOpenSpec verifies openspec is available. Required for plan command only.
func checkOpenSpec() error {
	return checkDep("openspec")
}
