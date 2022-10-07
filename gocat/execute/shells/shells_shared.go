package shells

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/mitre/gocat/execute"
	"github.com/mitre/gocat/output"
)

func checkExecutorInPath(path string) bool {
	_, err := exec.LookPath(path)
	output.VerbosePrint(fmt.Sprint(err))
	return err == nil
}

func runShellExecutor(cmd exec.Cmd, timeout int) (execute.CommandResults) {
	done := make(chan error, 1)
	status := execute.SUCCESS_STATUS
	var stdoutBuf, stderrBuf bytes.Buffer
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = getPlatformSysProcAttrs()
	}
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	executionTimestamp := time.Now().UTC()
	err := cmd.Start()
	if err != nil {
		errorBytes := []byte(fmt.Sprintf("Encountered an error starting the process: %q", err.Error()))
		return execute.CommandResults{[]byte{}, errorBytes, execute.ERROR_STATUS, execute.ERROR_PID, executionTimestamp}
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		err := cmd.Process.Kill()
		stdoutBytes := stdoutBuf.Bytes()
		stderrBytes := stderrBuf.Bytes()

		if err != nil {
			stderrBytes = append([]byte("Timeout reached, but couldn't kill the process\n"), stderrBytes...)
			return execute.CommandResults{stdoutBytes, stderrBytes, execute.ERROR_STATUS, pid, executionTimestamp}
		}
		stdoutBytes = append([]byte("Timeout reached, process killed\n"), stdoutBytes...)
		return execute.CommandResults{stdoutBytes, stderrBytes, execute.TIMEOUT_STATUS, pid, executionTimestamp}
	case err := <-done:
		stdoutBytes := stdoutBuf.Bytes()
		stderrBytes := stderrBuf.Bytes()
		if err != nil {
			status = execute.ERROR_STATUS
		}
		return execute.CommandResults{stdoutBytes, stderrBytes, status, pid, executionTimestamp}
	}
}
