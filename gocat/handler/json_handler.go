package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mitre/gocat/execute"
	"github.com/mitre/gocat/payload"
)

const name = "json-config"

type JsonCommandHandler struct {
	BaseCommandHandler
	shortName string
}

func init() {
	Handlers[name] = &JsonCommandHandler{
		shortName: name,
	}
}

//Run the actual command
func (ch *JsonCommandHandler) HandleCommand(info execute.InstructionInfo) ([]byte, string, string, time.Time) {
	// This layer is above executor and below agent.
	// 1) Parse instruction header for executor
	// 2) Dump behavior DSL into a JSON on disk (IDEA: consider support for CLi JSON passing too, or IPC (after startup))
	// 3) run the payload (parser + runner + action library) w/ the JSON
	// 4) add JSON file to payloads in InstructionInfo so it can be removed if the payload doesn't.

	var status string
	var result []byte
	var pid string

	err_handler := func(str string) ([]byte, string, string, time.Time) {
		result = []byte(str)
		status = execute.ERROR_STATUS
		pid = execute.ERROR_STATUS
		return result, status, pid, time.Now()
	}

	encodedCommand := info.Instruction["command"].(string)
	executor := info.Instruction["executor"].(string)
	timeout := int(info.Instruction["timeout"].(float64))

	missingPaths := payload.CheckIfOnDisk(info.OnDiskPayloads)
	if len(missingPaths) != 0 {
		return err_handler(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
	}

	ch.AddPayloads(info.OnDiskPayloads, info.InMemoryPayloads)

	decoded, err := base64.StdEncoding.DecodeString(encodedCommand)
	if err != nil {
		return err_handler(fmt.Sprintf("Error when decoding command: %s", err.Error()))
	}

	command := make(map[string]interface{}) // TODO marshal this into a JSON schema struct for validation
	err = json.Unmarshal([]byte(string(decoded)), &command)
	if err != nil {
		return err_handler(fmt.Sprintf("Error when unmarshaling command JSON: %s", err.Error()))
	}

	config_bytes, ok := command["config"].([]byte)
	if !ok {
		return err_handler(fmt.Sprintf("Error when getting config bytes: %s", err.Error()))
	}

	if location, err := payload.WriteToDisk("config", config_bytes); err != nil {
		return err_handler(fmt.Sprintf("Error writing config to disk: %s", err.Error()))
	} else {
		ch.AddDiskPayload(location)
	}

	header := command["header"].(map[string]interface{})
	run_cmd := header["start_command"].(string)

	return execute.Executors[executor].Run(run_cmd, timeout, info)
}