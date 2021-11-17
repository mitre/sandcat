package shells

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	"github.com/google/shlex"

	"github.com/mitre/gocat/execute"
	"github.com/mitre/gocat/output"
)

type Proc struct {
	name string
	currDir string
}

func init() {
	cwd, _ := os.Getwd()
    executor := &Proc{
		name: "proc",
		currDir: cwd,
	}
	execute.Executors[executor.name] = executor
}

func (p *Proc) Run(command string, timeout int, info execute.InstructionInfo) ([]byte, string, string, time.Time) {
	exePath, exeArgs, err := p.getExeAndArgs(command)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error parsing command line: %s", err.Error()))
		return nil, "", "", time.Now().UTC()
	}
	output.VerbosePrint(fmt.Sprintf("[*] Starting process %s with args %v", exePath, exeArgs))
	if exePath == "del" || exePath == "rm" {
		return p.deleteFiles(exeArgs)
	}
	return runShellExecutor(*exec.Command(exePath, append(exeArgs)...), timeout)
}

func (p *Proc) String() string {
	return p.name
}

func (p *Proc) CheckIfAvailable() bool {
	return true
}

func (p *Proc) DownloadPayloadToMemory(payloadName string) bool {
	return false
}

func (p *Proc) getExeAndArgs(commandLine string) (string, []string, error) {
	if runtime.GOOS == "windows" {
		commandLine = strings.ReplaceAll(commandLine, "\\", "\\\\")
	}
	split, err := shlex.Split(commandLine)
	if err != nil {
		return "", nil, err
	}
	return split[0], split[1:], nil
}

func (p *Proc) UpdateBinary(newBinary string) {
	return
}

func (p *Proc) deleteFiles(files []string) ([]byte, string, string, time.Time) {
	pid := strconv.Itoa(os.Getpid())
	var outputMessages []string
	var msg string
	var err error
	status := execute.SUCCESS_STATUS
	executionTimestamp := time.Now().UTC()
	for _, toDelete := range files {
		err = os.Remove(toDelete)
		if err != nil {
			msg = fmt.Sprintf("Failed to remove %s: %s", toDelete, err.Error())
			status = execute.ERROR_STATUS
		} else {
			msg = fmt.Sprintf("Removed file %s.", toDelete)
		}
		outputMessages = append(outputMessages, msg)
	}
	return []byte(strings.Join(outputMessages, "\n")), status, pid, executionTimestamp
}