// +build windows

package donut

import (
	"syscall"
	"unsafe"

	"github.com/mitre/sandcat/gocat/executors/execute"
	"github.com/mitre/sandcat/gocat/util"
)

// Runner runner
func Runner(donut []byte, handle syscall.Handle, stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte) (bool, string) {

	address, err := VirtualAllocEx(handle, 0, uintptr(len(donut)), MEM_COMMIT|MEM_RESERVE, syscall.PAGE_EXECUTE_READ)
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}

	var bytesWritten uintptr

	_, err = WriteProcessMemory(handle, address, (uintptr)(unsafe.Pointer(&donut[0])), uintptr(len(donut)), &bytesWritten)
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}

	var threadHandle syscall.Handle

	threadHandle, err = CreateRemoteThread(handle, nil, 0, address, 0, 0, 0)
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}

	_, err = WaitForSingleObject(threadHandle, 0xFFFFFFFF)
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}

	//Terminate the sacrificial process
	err = TerminateProcess(handle, 0)
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}

	err = ReadFromPipes(stdout, stdoutBytes, stderr, stderrBytes)
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}

	return true, execute.SUCCESS_PID
}

// IsAvailable does a donut runner exist
func IsAvailable() bool {

	return true
}