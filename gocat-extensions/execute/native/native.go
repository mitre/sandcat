// +build windows darwin linux

package native

import (
	"strings"
	"time"
	"os"
	"strconv"
	"fmt"

	"github.com/mitre/gocat/execute"
)

type Native struct {
	shortName string
	path string
}

func init() {
	shell := &Native {
		shortName: "native",
	}
	execute.Executors[shell.shortName] = shell
	fmt.Println("LOADED NATIVE")
}

func (n *Native) Run(command string, timeout int, info execute.InstructionInfo) ([]byte, string, string) {
	return n.runNativeExecutor(command, timeout)
}

func (n *Native) String() string {
	return n.shortName
}

func (n *Native) CheckIfAvailable() bool {
	return true
}

func (n *Native) runNativeExecutor(command string, timeout int) ([]byte, string, string) {
	pid := strconv.Itoa(os.Getpid())
	var cmd func(chan []byte, chan string)
	chOutput := make(chan []byte)
	chStatus := make(chan string)

	switch {
	case strings.EqualFold(command, "ip_addr"):
		cmd = getIPAddresses
	default:
		errorOutput := []byte("Invalid command: " + command)
		return errorOutput, execute.ERROR_STATUS, pid
	}

	go cmd(chOutput, chStatus)

	select {
	case cmdOutput := <-chOutput:
		status := <-chStatus
		return cmdOutput, status, pid
	case <-time.After(time.Duration(timeout) * time.Second):
		return []byte("Timeout reached running: " + command), execute.TIMEOUT_STATUS, pid
	}
}
