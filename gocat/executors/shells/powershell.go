// +build windows

package shells

import (
	"../execute"
	"os/exec"
)

type Powershell struct {
	shortName string
	path string
	execArgs []string
}

func init() {
	shell := &Powershell{
		shortName: "psh",
		path: "powershell.exe",
		execArgs: []string{"-ExecutionPolicy", "Bypass", "-C"},
	}
	if shell.CheckIfAvailable() {
		execute.Executors[shell.path] = shell
	}
}

func (p *Powershell) Run(command string, timeout int) ([]byte, string, string) {
	return runShellExecutor(*exec.Command(p.path, append(p.execArgs, command)...), timeout)
}

func (p *Powershell) String() string {
	return p.shortName
}

func (p *Powershell) CheckIfAvailable() bool {
	return checkExecutorInPath(p.path)
} 