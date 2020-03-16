package shells

import (
	"github.com/mitre/sandcat/gocat/executors/execute"
	"os/exec"
)

type Sh struct {
	path string
	execArgs []string
}

func init() {
	shell := &Sh{
		path: "sh",
		execArgs: []string{"-c"},
	}
	if shell.CheckIfAvailable() {
		execute.Executors[shell.path] = shell
	}
}

func (s *Sh) Run(command string, timeout int) ([]byte, string, string) {
	return runShellExecutor(*exec.Command(s.path, append(s.execArgs, command)...), timeout)
}

func (s *Sh) String() string {
	return s.path
}

func (s *Sh) CheckIfAvailable() bool {
	return checkExecutorInPath(s.path)
}