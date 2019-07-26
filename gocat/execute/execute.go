package execute

import (
	"runtime"
	"os/exec"
)

// Execute runs a shell command
func Execute(command string) ([]byte, error) {
	if runtime.GOOS == "windows" {
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).CombinedOutput()
	}
	return exec.Command("sh", "-c", command).CombinedOutput()
}
