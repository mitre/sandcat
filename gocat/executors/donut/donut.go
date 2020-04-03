// +build windows linux

package donut

import (
	"io/ioutil"
	"runtime"

	"../execute"
	"../../util"
)

type Donut struct {
	archName string
}

func init() {
	runner := &Donut{
		archName: "donut"+runtime.GOARCH,
	}
	if runner.CheckIfAvailable() {
		execute.Executors[runner.archName] = runner
	}
}

func (s *Donut) Run(command string, timeout int) ([]byte, string, string) {
	bytes, _ := ioutil.ReadFile("shellcode.bin")

	//gToByteArrayString(string(content))

	//bytes, _ := util.StringToByteArrayString(command)
	//task, pid := Runner(bytes)

	task, pid := Runner(bytes)
	if task {
		return []byte("Shellcode executed successfully."), execute.SUCCESS_STATUS, pid
	}
	return []byte("Shellcode execution failed."), execute.ERROR_STATUS, pid
}

func (s *Donut) String() string {
	return s.archName
}

func (s *Donut) CheckIfAvailable() bool {
	return IsAvailable()
}