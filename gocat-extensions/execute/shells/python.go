// +build windows darwin linux

package shells

import (
	"os/exec"
//	"runtime"
//	"fmt"

	"github.com/mitre/gocat/execute"
)

type Python struct {
	shortName string
	path string
	execArgs []string
}

func init() {
//	var path string

/*	var err string

	if runtime.GOOS == "windows" {
		path, err := exec.Command("powershell","where.exe","python.exe").Output()
		if path == "INFO: Could not find files for the given pattern(s)." {
		  //fmt.Println("Python is not installed\n")
//		} else if path {
            //path contains Python3*
		} else {
		    //fmt.Println("%s\n",path)
		}
// may not need
	} else if runtime.GOOS == "linux" {
	    //which python3
	    path, err := exec.LookPath("python3").Output()
	    if path == ""  {
	        //fmt.Println("Python3 is not installed\n")
		} else {
		    //fmt.Println("%s\n",path)
		}
	} else {
		path = "python3"
	}
*/

	shell := &Python{
		shortName: "python3",
		path: "python3",
		execArgs: []string{"-c"},
	}
	if shell.CheckIfAvailable() {
		execute.Executors[shell.shortName] = shell
	}
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