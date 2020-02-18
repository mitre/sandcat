package shells

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"../execute"
	"../../output"
)

func checkExecutorInPath(path string) bool {
	_, err := exec.LookPath(path)
	output.VerbosePrint(fmt.Sprint(err))
	return err == nil
}

func runShellExecutor(cmd exec.Cmd, timeout int) ([]byte, string, string) {
	done := make(chan error, 1)
	status := execute.SUCCESS_STATUS
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SysProcAttr = getPlatformSysProcAttrs()
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Start()
	if err != nil {
		return []byte(fmt.Sprintf("Encountered an error starting the process: %q", err.Error())), execute.ERROR_STATUS, execute.ERROR_PID
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return []byte("Timeout reached, but couldn't kill the process"), execute.ERROR_STATUS, pid
		}
		return []byte("Timeout reached, process killed"), execute.TIMEOUT_STATUS, pid
	case err := <-done:
		stdoutBytes := stdoutBuf.Bytes()
		stderrBytes := stderrBuf.Bytes()
		if err != nil {
			status = execute.ERROR_STATUS
		}
		if len(stderrBytes) > 0 {
			return stderrBytes, status, pid
		}
		return stdoutBytes, status, pid
	}
}