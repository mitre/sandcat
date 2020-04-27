// +build windows

package donut

import (
	"fmt"
	"io/ioutil"
	"runtime"

	"github.com/mitre/gocat/execute"
)

type Donut struct {
	archName string
}

func init() {
	runner := &Donut{
		archName: "donut_" + runtime.GOARCH,
	}
	if runner.CheckIfAvailable() {
		execute.Executors[runner.archName] = runner
	}
}

const COMMANDLINE = "rundll32.exe"

func (d *Donut) Run(command string, timeout int) ([]byte, string, string) {
	bytes, _ := ioutil.ReadFile(command)

	handle, pid, stdout, stderr := CreateSuspendedProcessWithIORedirect("rundll32.exe")

	// Setup variables
	stdoutBytes := make([]byte, 1)
	stderrBytes := make([]byte, 1)
	var eventCode uint32

	// Run the shellcode and wait for it to complete
	task, err := Runner(bytes, handle, stdout, &stdoutBytes, stderr, &stderrBytes, &eventCode)

	// Assemble the final output
	if task {

		total := "Shellcode thread Exit Code: " + fmt.Sprint(eventCode) + "\n\n"

		total += "STDOUT:\n"
		total += string(stdoutBytes)
		total += "\n\n"

		total += "STDERR:\n"
		total += string(stderrBytes)

		return []byte(total), execute.SUCCESS_STATUS, fmt.Sprint(pid)
	}

	// Covers the cases where an error was received before the remote thread was created
	return []byte(fmt.Sprintf("Shellcode execution failed. Error message: %s", fmt.Sprint(err))), execute.ERROR_STATUS, fmt.Sprint(pid)
}

func (d *Donut) String() string {
	return d.archName
}

func (d *Donut) CheckIfAvailable() bool {
	return IsAvailable()
}
