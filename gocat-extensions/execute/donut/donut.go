// +build windows

package donut

import (
	"fmt"
	"strings"
	"runtime"

	"github.com/mitre/gocat/execute"
	"github.com/mitre/gocat/output"
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

const COMMANDLINE string = "rundll32.exe"

func (d *Donut) Run(command string, timeout int, info execute.InstructionInfo) ([]byte, string, string) {
    // Setup variables
    stdoutBytes := make([]byte, 1)
    stderrBytes := make([]byte, 1)
    var eventCode uint32
    inMemoryPayloads := info.InMemoryPayloads
    payload, bytes := getDonutBytes(inMemoryPayloads)

    if bytes != nil && len(bytes) > 0 {
        // Create sacrificial process
        output.VerbosePrint(fmt.Sprintf("[i] Donut: Creating sacrificial process '%s'", COMMANDLINE))
        handle, pid, stdout, stderr := CreateSuspendedProcessWithIORedirect(COMMANDLINE)
        output.VerbosePrint(fmt.Sprintf("[i] Donut: Created sacrificial process with PID %d", pid))

        // Run the shellcode and wait for it to complete
        output.VerbosePrint(fmt.Sprint("[i] Donut: Running shellcode"))
        task, err := Runner(bytes, handle, stdout, &stdoutBytes, stderr, &stderrBytes, &eventCode)
        output.VerbosePrint(fmt.Sprint("[i] Donut: Shellcode execution finished"))

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
    } else {
        // Empty payload
        return []byte(fmt.Sprintf("Empty payload: %s", payload)), execute.ERROR_STATUS, "-1"
    }
}

func (d *Donut) String() string {
	return d.archName
}

func (d *Donut) CheckIfAvailable() bool {
	return IsAvailable()
}

func (d* Donut) DownloadPayloadToMemory(payloadName string) bool {
	return strings.HasSuffix(payloadName, ".donut")
}

// Since donut abilities only require one payload, grab the first in-memory payload available.
func getDonutBytes(inMemoryPayloads map[string][]byte) (string, []byte) {
	for payloadName, payloadBytes := range inMemoryPayloads {
		return payloadName, payloadBytes
	}
	return "", nil
}