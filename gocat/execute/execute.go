package execute

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"../shellcode"
)

// ExecutorFlags type to import a list of executors
type ExecutorFlags []string

// Execute runs a shell command
func Execute(command string, executor string) ([]byte, error) {
	if command == "die" {
		return []byte("shutdown started"), nil
	}

	if executor == "psh" {
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).CombinedOutput()
	} else if executor == "cmd" {
		return exec.Command("cmd", "/C", command).CombinedOutput()
	} else if executor == "pwsh" {
		return exec.Command("pwsh", "-c", command).CombinedOutput()
	} else if executor == fmt.Sprintf("shellcode_%s", runtime.GOARCH) {
		return shellcode.ExecuteShellcode(command)
	}
	return exec.Command("sh", "-c", command).CombinedOutput()
}

// DetermineExecutor executor type, using sane defaults
func DetermineExecutor(executors []string, platform string, arch string) []string {
	if executors == nil {
		if platform == "windows" {
			executors = append(executors, "psh")
		} else {
			executors = append(executors, "sh")
		}
	}
	return checkShellcodeExecutors(executors, arch)
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

// CheckShellcodeExecutors checks if shellcode execution is available
func checkShellcodeExecutors(executors []string, arch string) []string {
	if shellcode.IsAvailable() {
		executors = append(executors, "shellcode_"+arch)
	}
	return executors
}
