// +build windows linux

package shellcode

import (
	"runtime"

	"github.com/mitre/sandcat/gocat/executors/execute"
	"github.com/mitre/sandcat/gocat/util"
)

type Shellcode struct {
	archName string
}

func init() {
	runner := &Shellcode{
		archName: "shellcode_"+runtime.GOARCH,
	}
	if runner.CheckIfAvailable() {
		execute.Executors[runner.archName] = runner
	}
}

func (s *Shellcode) Run(command string, timeout int) ([]byte, string, string) {
	bytes, _ := util.StringToByteArrayString(command)
	task, pid := Runner(bytes)
	if task {
		return []byte("Shellcode executed successfully."), execute.SUCCESS_STATUS, pid
	}
	return []byte("Shellcode execution failed."), execute.ERROR_STATUS, pid
}

func (s *Shellcode) String() string {
	return s.archName
}

func (s *Shellcode) CheckIfAvailable() bool {
	return IsAvailable()
}