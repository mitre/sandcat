// +build windows darwin linux

package shells

import (
	"os/exec"
	"runtime"
	"fmt"
	"strings"
	"github.com/mitre/gocat/output"
	"github.com/mitre/gocat/execute"
)

type Python struct {
	shortName string
	path string
	execArgs []string
}

func init() {
	setExecutor("python3")
	setExecutor("python")
}

func setExecutor(name string) bool {
// Checks if python3 or python is available on the system and
// sets the shell executor with the appropriate path
	var path string
	var val string
	var sName string
	
	if runtime.GOOS == "windows" {
		path = name + ".exe"
	} else {
		path = name
	}

	if name == "python" {
		val = checkVersion(name)
		sName = "python" + val
	} else {
		sName = name
	}

	shell := &Python {
		shortName: sName,
		path: path,
		execArgs: []string{"-c"},
	}
	if shell.CheckIfAvailable() {
		execute.Executors[shell.shortName] = shell
		return true
	} 
	fmt.Print("%s is not installed", name)
	return false
}

func checkVersion(name string) string {
	var str_ver string
// checks the python version 
	version, err := exec.Command("python", "-c", "import platform; print(platform.python_version().split('.')[0])").CombinedOutput()
	fmt.Print("version", string(version), "\n")
	if err != nil {
		fmt.Print("Error:", err, "\n")
	}
// returns the first number from the python version (i.e. python version 2.7.1 would return 2 as a string)
	str_ver = string(version)
	str_ver = strings.TrimSuffix(str_ver, "\n")
	str_ver = strings.TrimSuffix(str_ver, "\r")
	return str_ver
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

func (p *Python) DownloadPayloadToMemory(payloadName string) bool {
	return false
}