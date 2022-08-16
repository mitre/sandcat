// +build windows linux

package shellcode

import (
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
	"unicode"
	"time"

	"github.com/mitre/gocat/execute"
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
	} else {
		fmt.Printf("Shellcode executor %s not available", runner.archName)
	}
}

func (s *Shellcode) Run(command string, timeout int, info execute.InstructionInfo) (execute.CommandResults) {
	bytes, _ := stringToByteArrayString(command)
	executionTimestamp := time.Now().UTC()
	task, pid := Runner(bytes)
	if task {
		return execute.CommandResults{[]byte("Shellcode executed successfully."), []byte{}, execute.SUCCESS_STATUS, pid, executionTimestamp}
	}
	return execute.CommandResults{[]byte{}, []byte("Shellcode execution failed."), execute.ERROR_STATUS, pid, executionTimestamp}
}

func (s *Shellcode) String() string {
	return s.archName
}

func (s *Shellcode) CheckIfAvailable() bool {
	return IsAvailable()
}

func (s *Shellcode) DownloadPayloadToMemory(payloadName string) bool {
	return false
}

func (s *Shellcode) UpdateBinary(newBinary string) {
	return
}

func stringToByteArrayString(input string) ([]byte, error) {
	temp := removeWhiteSpace(input)
	temp = strings.Replace(temp, "0x", "", -1)
	temp = strings.Replace(temp, "\\x", "", -1)
	temp = strings.Replace(temp, ",", "", -1)
	return hex.DecodeString(temp)
}

func removeWhiteSpace(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, input)
}
