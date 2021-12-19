package execute

import "time"

const (
	SUCCESS_STATUS = "0"
	ERROR_STATUS   = "1"
	TIMEOUT_STATUS = "124"
	SUCCESS_PID    = "0"
	ERROR_PID      = "1"
)

var Executors = map[string]Executor{}

type Executor interface {
	// Run takes a command string, timeout int, and instruction info.
	// Returns Raw Output, A String status code, and a String PID
	Run(command string, timeout int, info InstructionInfo) ([]byte, string, string, time.Time)
	String() string
	CheckIfAvailable() bool
	UpdateBinary(newBinary string)

	// Returns true if the executor wants the payload downloaded to memory, false if it wants the payload on disk.
	DownloadPayloadToMemory(payloadName string) bool
}

type InstructionInfo struct {
	Profile          map[string]interface{}
	Instruction      map[string]interface{}
	OnDiskPayloads   []string
	InMemoryPayloads map[string][]byte
}

func AvailableExecutors() (values []string) {
	names := make([]string, 0, len(Executors))
	for key := range Executors {
		names = append(names, key)
	}
	return names
}

func RemoveExecutor(name string) {
	delete(Executors, name)
}
