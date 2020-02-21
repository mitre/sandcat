package execute

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"../shellcode"
	"../util"
)

const (
	SUCCESS_STATUS = "0"
	ERROR_STATUS = "1"
	TIMEOUT_STATUS = "124"
)

// ExecutorFlags type to import a list of executors
type ExecutorFlags []string

//RunCommand runs the actual command
func RunCommand(command string, payloads []string, platform string, executor string, timeout int) ([]byte, string, string){
	cmd := string(util.Decode(command))
	var status string
	var result []byte
	var pid string
	missingPaths := util.CheckPayloadsAvailable(payloads)
	if len(missingPaths) == 0 {
		result, status, pid = Execute(cmd, executor, platform, timeout)
	} else {
		result = []byte(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
		status = ERROR_STATUS
		pid = ERROR_STATUS
	}
	return result, status, pid
}

// Execute runs a shell command
func Execute(command string, executor string, platform string, timeout int) ([]byte, string, string) {
	var output []byte
	var err error
	var pid string
	status := SUCCESS_STATUS
	if executor == fmt.Sprintf("shellcode_%s", runtime.GOARCH) {
		output, err, pid = shellcode.ExecuteShellcode(command)
		if err != nil {
			status = ERROR_STATUS
		}
		return output, status, pid
	}
	return runShellExecutor(executor, platform, command, timeout)
}

// DetermineExecutor executor type, using sane defaults
func DetermineExecutor(executors []string, platform string, arch string) []string {
	platformExecutors := map[string]map[string][]string {
		"windows": {
			"file": {"powershell.exe", "cmd.exe", "pwsh.exe"},
			"executor": {"psh", "cmd", "pwsh"},
		},
		"linux": {
			"file": {"sh", "pwsh"},
			"executor": {"sh", "pwsh"},
		},
		"darwin": {
			"file": {"sh", "pwsh", "osascript"},
			"executor": {"sh", "pwsh", "osa"},
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
	switch executor {
	case "psh":
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command)
	case "cmd":
		return exec.Command("cmd.exe", "/C", command)
	case "pwsh":
		if platform == "windows" {
			return exec.Command("pwsh.exe", "-c", command)
		} else {
			return exec.Command("pwsh", "-c", command)
		}
	case "osa" :
		return exec.Command("osascript", "-e", command)
	default:
		return exec.Command("sh", "-c", command)
	}
}

func runShellExecutor(executor string, platform string, command string, timeout int) ([]byte, string, string) {
	done := make(chan error, 1)
	status := SUCCESS_STATUS
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := buildCommandStatement(executor, platform, command)
	cmd.SysProcAttr = getPlatformSysProcAttrs()
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Start()
	if err != nil {
		return []byte(fmt.Sprintf("Encountered an error starting the process: %q", err.Error())), ERROR_STATUS, shellcode.ERROR_PID
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return []byte("Timeout reached, but couldn't kill the process"), ERROR_STATUS, pid
		}
		return []byte("Timeout reached, process killed"), TIMEOUT_STATUS, pid
	case err := <-done:
		stdoutBytes := stdoutBuf.Bytes()
		stderrBytes := stderrBuf.Bytes()
		if err != nil {
			status = ERROR_STATUS
		}
		if len(stderrBytes) > 0 {
			return stderrBytes, status, pid
		}
		return stdoutBytes, status, pid
	}
}
