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
func Execute(command string, executor string, platform string, resultChan chan map[string]interface{}) {
	if command == "die" {
		resultChan <- map[string]interface{}{"result":[]byte("shutdown started"), "err": nil}
	}
	var output []byte
	var err error
	if executor == "psh" {
		output, err = exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).CombinedOutput()
	} else if executor == "cmd" {
		output, err = exec.Command("cmd.exe", "/C", command).CombinedOutput()
	} else if platform == "windows" && executor == "pwsh" {
		output, err = exec.Command("pwsh.exe", "-c", command).CombinedOutput()
	} else if executor == "pwsh" {
		output, err = exec.Command("pwsh", "-c", command).CombinedOutput()
	} else if executor == fmt.Sprintf("shellcode_%s", runtime.GOARCH) {
		output, err = shellcode.ExecuteShellcode(command)
	} else {
		output, err = exec.Command("sh", "-c", command).CombinedOutput()
	}
	resultChan <- map[string]interface{}{"result":output, "err":err}
}

// DetermineExecutor executor type, using sane defaults
func DetermineExecutor(executors []string, platform string, arch string) []string {
	if executors == nil {
		if platform == "windows" {
			if checkIfExecutorAvailable("powershell.exe") {
				executors = append(executors, "psh")
			}
			if checkIfExecutorAvailable("pwsh.exe") {
				executors = append(executors, "pwsh")
			}
			if checkIfExecutorAvailable("cmd.exe") {
				executors = append(executors, "cmd")
			}
		} else {
			if checkIfExecutorAvailable("sh") {
				executors = append(executors, "sh")
			}
			if checkIfExecutorAvailable("pwsh") {
				executors = append(executors, "pwsh")
			}
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

func checkShellcodeExecutors(executors []string, arch string) []string {
	if shellcode.IsAvailable() {
		executors = append(executors, "shellcode_"+arch)
	}
	return executors
}

func checkIfExecutorAvailable(executor string) bool {
	_, err := exec.LookPath(executor)
	return err == nil
}