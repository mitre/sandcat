package execute

import (
	"os/exec"

	"../shellcode"
)

// Execute runs a shell command
func Execute(command string, executor string) ([]byte, error) {
	if executor == "psh" {
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).CombinedOutput()
	} else if executor == "cmd" {
		return exec.Command(command).CombinedOutput()
	} else if executor == "shellcode_x64" {
		return shellcode.ExecuteShellcode(command)
	}
	return exec.Command("sh", "-c", command).CombinedOutput()
}

// DetermineExecutor executor type, using sane defaults
func DetermineExecutor(platform string) string {
	if platform == "windows" {
		return "psh"
	}
	return "sh"
}
