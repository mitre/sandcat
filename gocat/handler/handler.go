package handler

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/mitre/gocat/execute"
	"github.com/mitre/gocat/payload"
)

const DefaultName = "base"

var Handlers = map[string]CommandHandler{}

type CommandHandler interface {
	HandleCommand(info execute.InstructionInfo) ([]byte, string, string, time.Time)
	AddPayloads(onDisk []string, inMemory map[string][]byte)
	GetPayloads() ([]string, map[string][]byte)
	String() string
	//GetStatus() string
}

func init() {
	Handlers[DefaultName] = &BaseCommandHandler{
		shortName:    DefaultName,
		diskPayloads: make(map[string]struct{}),
		memPayloads:  make(map[string][]byte),
	}
}

type BaseCommandHandler struct {
	shortName    string
	diskPayloads map[string]struct{}
	memPayloads  map[string][]byte
}

//Run the actual command
func (ch *BaseCommandHandler) HandleCommand(info execute.InstructionInfo) ([]byte, string, string, time.Time) {
	encodedCommand := info.Instruction["command"].(string)
	executor := info.Instruction["executor"].(string)
	timeout := int(info.Instruction["timeout"].(float64))
	onDiskPayloads := info.OnDiskPayloads
	var status string
	var result []byte
	var pid string
	var executionTimestamp time.Time
	decoded, err := base64.StdEncoding.DecodeString(encodedCommand)
	if err != nil {
		result = []byte(fmt.Sprintf("Error when decoding command: %s", err.Error()))
		status = execute.ERROR_STATUS
		pid = execute.ERROR_STATUS
		executionTimestamp = time.Now()
	} else {
		command := string(decoded)
		missingPaths := payload.CheckIfOnDisk(onDiskPayloads)
		if len(missingPaths) == 0 {
			result, status, pid, executionTimestamp = execute.Executors[executor].Run(command, timeout, info)
		} else {
			result = []byte(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
			status = execute.ERROR_STATUS
			pid = execute.ERROR_STATUS
			executionTimestamp = time.Now()
		}
	}
	return result, status, pid, executionTimestamp
}

func (ch *BaseCommandHandler) String() string {
	return ch.shortName
}

func (ch *BaseCommandHandler) AddDiskPayload(name string) {
	ch.diskPayloads[name] = struct{}{}
}

func (ch *BaseCommandHandler) AddMemoryPayload(name string, data []byte) {
	ch.memPayloads[name] = data
}

func (ch *BaseCommandHandler) AddPayloads(onDisk []string, inMemory map[string][]byte) {
	for _, v := range onDisk {
		ch.AddDiskPayload(v)
	}
	for k, v := range inMemory {
		ch.AddMemoryPayload(k, v)
	}
}

func (ch *BaseCommandHandler) GetPayloads() ([]string, map[string][]byte) {
	diskPayloadNames := make([]string, 0, len(ch.diskPayloads))
	for key := range ch.diskPayloads {
		diskPayloadNames = append(diskPayloadNames, key)
	}
	return diskPayloadNames, ch.memPayloads
}

func AvailableCommandHandlers() (values []string) {
	names := make([]string, 0, len(Handlers))
	for key := range Handlers {
		names = append(names, key)
	}
	return names
}
