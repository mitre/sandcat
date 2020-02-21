package execute

import (
	"../../util"
	"fmt"
	"strings"
)

const (
	SUCCESS_STATUS 	= "0"
	ERROR_STATUS 	= "1"
	TIMEOUT_STATUS 	= "124"
	SUCCESS_PID 	= "0"
	ERROR_PID 	= "1"
)

type Executor interface {
	// Run takes an arbitrary command string & timeout int, Returns Raw Output, A String status code, and a String PID
	Run(command string, timeout int) ([]byte, string, string)
	String() string
	CheckIfAvailable() bool
}

func AvailableExecutors() (values []string) {
	for _, e := range Executors {
		values = append(values, e.String())
	}
	return
}

var Executors = map[string]Executor{}

//RunCommand runs the actual command
func RunCommand(command string, payloads []string, executor string, timeout int) ([]byte, string, string){
	cmd := string(util.Decode(command))
	var status string
	var result []byte
	var pid string
	missingPaths := util.CheckPayloadsAvailable(payloads)
	if len(missingPaths) == 0 {
		result, status, pid = Executors[executor].Run(cmd, timeout)
	} else {
		result = []byte(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
		status = ERROR_STATUS
		pid = ERROR_STATUS
	}
	return result, status, pid
}