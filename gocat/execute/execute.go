package execute

import (
	"../shellcode"
	"../util"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ExecutorFlags type to import a list of executors
type ExecutorFlags []string

// Execute runs a shell command
func Execute(command string, executor string) ([]byte, error) {
	if command == "die" {
		executable, _ := os.Executable()

		if executor == "sh" {
			util.DeleteFile(executable)
		} else {
			_, _ = exec.Command("cmd", "/C", "start", "cmd.exe", "/C", "timeout 1 & del C:\\Users\\Public\\sandcat.exe").CombinedOutput()
		}
		util.StopProcess(os.Getppid())
		util.StopProcess(os.Getpid())
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
