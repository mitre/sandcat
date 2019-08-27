package execute

import (
	"os"
	"os/exec"
)

// Execute runs a shell command
func Execute(command string, executor string) ([]byte, error) {
	if command == "die" {
		ppid := os.Getppid()
		proc, _ := os.FindProcess(ppid)
		_ = proc.Kill()
	}

	if executor == "psh" {
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).CombinedOutput()
	} else if executor == "cmd" {
		return exec.Command(command).CombinedOutput()
	} else if executor == "pwsh" {
		return exec.Command("pwsh", "-c", command).CombinedOutput()
	}
	return exec.Command("sh", "-c", command).CombinedOutput()
}

// DetermineExecutor executor type, using sane defaults
func DetermineExecutor(platform string) string {
	if platform == "windows" {
		return "psh"
	} else {
		return "sh"
	}
}