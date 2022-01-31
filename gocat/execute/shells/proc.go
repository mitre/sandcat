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

type CwdGetter func() (string, error)
type OsGetter func() string
type PidGetter func() int
type FileDeleter func(string) error
type TimeGenerator func() time.Time

type Proc struct {
	name string
	currDir string
	osName string
	pidStr string
	fileDeleter FileDeleter
	timeStampGenerator TimeGenerator
}

func getOsName() string {
	return runtime.GOOS
}

func getUtcTime() time.Time {
	return time.Now().UTC()
}

type ProcFunctionHandles struct {
	cwdGetter CwdGetter
	osGetter OsGetter
	pidGetter PidGetter
	fileDeleter FileDeleter
	timeStampGenerator TimeGenerator
}

func GenerateProcExecutor(funcHandles *ProcFunctionHandles) *Proc {
	cwd, _ := funcHandles.cwdGetter()
	osName := funcHandles.osGetter()
	pid := funcHandles.pidGetter()
	return &Proc{
		name: "proc",
		currDir: cwd,
		osName: osName,
		pidStr: strconv.Itoa(pid),
		fileDeleter: funcHandles.fileDeleter,
		timeStampGenerator: funcHandles.timeStampGenerator,
	}
}

func init() {
	procFuncHandles := &ProcFunctionHandles{
		cwdGetter: os.Getwd,
		osGetter: getOsName,
		pidGetter: os.Getpid,
		fileDeleter: os.Remove,
		timeStampGenerator: getUtcTime,
	}
	executor := GenerateProcExecutor(procFuncHandles)
	execute.Executors[executor.name] = executor
}

func (p *Proc) Run(command string, timeout int, info execute.InstructionInfo) ([]byte, string, string, time.Time) {
	exePath, exeArgs, err := p.getExeAndArgs(command)
	if err != nil {
		errMsg := fmt.Sprintf("[!] Error parsing command line: %s", err.Error())
		output.VerbosePrint(errMsg)
		return []byte(errMsg), execute.ERROR_STATUS, execute.ERROR_PID, p.timeStampGenerator()
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
	if p.osName == "windows" {
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
	var outputMessages []string
	var msg string
	var err error
	status := execute.SUCCESS_STATUS
	executionTimestamp := p.timeStampGenerator()
	for _, toDelete := range files {
		err = p.fileDeleter(toDelete)
		if err != nil {
			msg = fmt.Sprintf("Failed to remove %s: %s", toDelete, err.Error())
			status = execute.ERROR_STATUS
		} else {
			msg = fmt.Sprintf("Removed file %s.", toDelete)
		}
		outputMessages = append(outputMessages, msg)
	}
	return []byte(strings.Join(outputMessages, "\n")), status, p.pidStr, executionTimestamp
}