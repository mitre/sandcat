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
func Execute(command string, executor string, resultChan chan []byte, errorChan chan bool) {
	if command == "die" {
		resultChan <- []byte("shutdown started")
	}
	var output []byte
	var err error
	if executor == "psh" {
		output, err = exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).CombinedOutput()
	} else if executor == "cmd" {
		output, err = exec.Command("cmd", "/C", command).CombinedOutput()
	} else if executor == "pwsh" {
		output, err = exec.Command("pwsh", "-c", command).CombinedOutput()
	} else if executor == fmt.Sprintf("shellcode_%s", runtime.GOARCH) {
		output, err = shellcode.ExecuteShellcode(command)
	}
	output, err = exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		<-errorChan
	}
	resultChan <- output
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
