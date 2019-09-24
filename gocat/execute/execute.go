package execute

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"../shellcode"
	"../util"
)

const (
	// TIMEOUT in seconds represents how long a single command should run before timing out
	TIMEOUT = 60
	SUCCESS_STATUS = "0"
	ERROR_STATUS = "1"
	TIMEOUT_STATUS = "124"
)

// ExecutorFlags type to import a list of executors
type ExecutorFlags []string

//RunCommand runs the actual command
func RunCommand(command string, payloads []string, platform string, executor string) (string, []byte, string){
	cmd := string(util.Decode(command))
	var status string
	var result []byte
	missingPaths := util.CheckPayloadsAvailable(payloads)
	if len(missingPaths) == 0 {
		result, status = Execute(cmd, executor, platform)
	} else {
		result = []byte(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
		status = ERROR_STATUS
	}
	return cmd, result, status
}

// Execute runs a shell command
func Execute(command string, executor string, platform string) ([]byte, string) {
	var output []byte
	var err error
	status := SUCCESS_STATUS
	if command == "die" {
		return []byte("shutdown started"), SUCCESS_STATUS
	}
	if executor == fmt.Sprintf("shellcode_%s", runtime.GOARCH) {
		output, err = shellcode.ExecuteShellcode(command)
		if err != nil {
			status = ERROR_STATUS
		}
		return output, status
	}
	return runShellExecutor(executor, platform, command)
}

// DetermineExecutor executor type, using sane defaults
func DetermineExecutor(executors []string, platform string, arch string) []string {
	platformExecutors := map[string]map[string][]string {
		"windows": {
			"file": {"cmd.exe", "powershell.exe", "pwsh.exe"},
			"executor": {"cmd", "psh", "pwsh"},
		},
		"linux": {
			"file": {"sh", "pwsh"},
			"executor": {"sh", "pwsh"},
		},
		"darwin": {
			"file": {"sh", "pwsh"},
			"executor": {"sh", "pwsh"},
		},
	}
	if executors == nil {
		for platformKey, platformValue := range platformExecutors {
			if platform == platformKey {
				for i := range platformValue["file"] {
					if checkIfExecutorAvailable(platformValue["file"][i]) {
						executors = append(executors, platformExecutors[platformKey]["executor"][i])
					}
				}
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

func buildCommandStatement(executor string, platform string, command string) *exec.Cmd {
	if executor == "psh" {
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command)
	} else if executor == "cmd" {
		return exec.Command("cmd.exe", "/C", command)
	} else if platform == "windows" && executor == "pwsh" {
		return exec.Command("pwsh.exe", "-c", command)
	} else if executor == "pwsh" {
		return exec.Command("pwsh", "-c", command)
	} else {
		return exec.Command("sh", "-c", command)
	}
}

func runShellExecutor(executor string, platform string, command string) ([]byte, string) {
	done := make(chan error, 1)
	status := SUCCESS_STATUS
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := buildCommandStatement(executor, platform, command)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return []byte("Encountered an error starting the process!"), ERROR_STATUS
	}
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(TIMEOUT * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return []byte("Timeout reached, but couldn't kill the process"), ERROR_STATUS
		}
		return []byte("Timeout reached, process killed"), TIMEOUT_STATUS
	case err := <-done:
		stdoutBytes := stdoutBuf.Bytes()
		stderrBytes := stderrBuf.Bytes()
		if err != nil {
			status = ERROR_STATUS
		}
		return append(stdoutBytes, stderrBytes...), status
	}
}