package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mitre/gocat/execute"
	"github.com/mitre/gocat/handler"
	"github.com/mitre/gocat/payload"
)

const name = "dab-dsl"

type ConfigCommandHandler struct {
	handler.BaseCommandHandler
	shortName string
}

func init() {
	handler.Handlers[name] = &ConfigCommandHandler{
		shortName: name,
	}
}

//Run the actual command
func (ch *ConfigCommandHandler) HandleCommand(info execute.InstructionInfo) ([]byte, string, string, time.Time) {
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
	onDiskPayloads := info.OnDiskPayloads

	decoded, err := base64.StdEncoding.DecodeString(encodedCommand)
	if err != nil {
		return err_handler(fmt.Sprintf("Error when decoding command: %s", err.Error()))
	}

	command := make(map[string]interface{}) // TODO marshal this into a JSON schema struct for validation
	err = json.Unmarshal([]byte(string(decoded)), &command)
	if err != nil {
		return err_handler(fmt.Sprintf("Error when unmarshaling command JSON: %s", err.Error()))
	}

	// Write behavior DSL to disk and add it to payloads.
	_, err = WriteConfigToDisk(command["behavior"])
	if err != nil {
		return err_handler(fmt.Sprintf("Error when writing DSL to disk: %s", err.Error()))
	}

	missingPaths := payload.CheckIfOnDisk(onDiskPayloads)
	if len(missingPaths) != 0 {
		return err_handler(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
	}

	header := command["header"].(map[string]interface{})
	run_cmd := header["start_command"].(string)

	return execute.Executors[executor].Run(run_cmd, timeout, info)
}

func WriteConfigToDisk(data interface{}, filename_opt ...string) (string, error) {
	// Save DSL to JSON file on disk
	// FIXME: Need to figure out best way to name this
	filename := "config"
	if len(filename_opt) > 0 {
		filename = filename_opt[0]
	}
	if file, err := json.MarshalIndent(data, "", " "); err != nil {
		return "", err
	} else {
		return payload.WriteToDisk(filename, file)
	}
}
