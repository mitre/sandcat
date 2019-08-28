package execute

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecutorFlags type to import a list of executors
type ExecutorFlags []string

// Execute runs a shell command
func Execute(command string, executor string) ([]byte, error) {
	if executor == "psh" {
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).CombinedOutput()
	} else if executor == "cmd" {
		return exec.Command("cmd", "/C", command).CombinedOutput()
	} else if executor == "pwsh" {
		return exec.Command("pwsh", "-c", command).CombinedOutput()
	}
	return exec.Command("sh", "-c", command).CombinedOutput()
}

// DetermineExecutor executor type, using sane defaults
func DetermineExecutor(platform string) string {
	if platform == "windows" {
		return "psh"
	}
	return "sh"
}

// String get string format of input
func (i *ExecutorFlags) String() string {
	return fmt.Sprint((*i))
}

// Set value of the executor list
func (i *ExecutorFlags) Set(value string) error {
	for _, exec := range strings.Split(value, ",") {
		*i = append(*i, exec)
	}
	return nil
}
