// +build windows darwin linux

package shells

import (
	"os/exec"
	"runtime"
	"fmt"
	"github.com/mitre/gocat/output"
	"github.com/mitre/gocat/execute"
)

type Python struct {
	shortName string
	path string
	execArgs []string
}

func init() {
	if !(setPath("python3")) {
		setPath("python")
	}
}

func setPath(name string) bool {
// Checks if python3 or python is available on the system and
//sets the shell executor with the appropriate path
	var path string

	if runtime.GOOS == "windows" {
		path = name + ".exe"
	} else {
		path = name
	}

	shell := &Python {
		shortName: "python3",
		path: path,
		execArgs: []string{"-c"},
	}
	if shell.CheckIfAvailable() {
		execute.Executors[shell.shortName] = shell
		return true
	} 
	output.VerbosePrint(fmt.Sprintf("%s is not installed", name))
	return false
}


func (p *Python) Run(command string, timeout int, info execute.InstructionInfo) ([]byte, string, string) {
	return runShellExecutor(*exec.Command(p.path, append(p.execArgs, command)...), timeout)
}

func (p *Python) String() string {
	return p.shortName
}

func (p *Python) CheckIfAvailable() bool {
	return checkExecutorInPath(p.path)
} 