// +build windows

package donut

import (
	"fmt"
	"log"
	"syscall"
	"time"
	"unsafe"

	"github.com/mitre/gocat/output"
)

const (
	MEM_COMMIT  = 0x1000
	MEM_RESERVE = 0x2000

	CREATE_SUSPENDED = 0x4
	CREATE_NO_WINDOW = 0x08000000

	SW_HIDE = 0
)

func CreateSuspendedProcessWithIORedirect(commandLine string) (syscall.Handle, uint32, syscall.Handle, syscall.Handle) {

	// Create anonymous pipe for STDOUT
	var stdOutRead syscall.Handle
	var stdOutWrite syscall.Handle

	errStdOutPipe := syscall.CreatePipe(&stdOutRead, &stdOutWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	errStdOutHandle := syscall.SetHandleInformation(stdOutRead, syscall.HANDLE_FLAG_INHERIT, 0)
	if errStdOutPipe != nil && errStdOutHandle != nil {
		output.VerbosePrint(fmt.Sprintf("[!]Error creating the STDOUT pipe:\r\n%s", errStdOutPipe.Error()))
	}

	// Create anonymous pipe for STDERR
	var stdErrRead syscall.Handle
	var stdErrWrite syscall.Handle

	errStdErrPipe := syscall.CreatePipe(&stdErrRead, &stdErrWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	errStdErrHandle := syscall.SetHandleInformation(stdErrRead, syscall.HANDLE_FLAG_INHERIT, 0)
	if errStdErrPipe != nil && errStdErrHandle != nil {
		output.VerbosePrint(fmt.Sprintf("[!]Error creating the STDERR pipe:\r\n%s", errStdErrPipe.Error()))
	}

	procInfo := &syscall.ProcessInformation{}
	startupInfo := &syscall.StartupInfo{
		StdOutput:  stdOutWrite,
		StdErr:     stdErrWrite,
		Flags:      syscall.STARTF_USESTDHANDLES | CREATE_SUSPENDED,
		ShowWindow: SW_HIDE,
	}

	errCreateProcess := CreateProcess(nil,
		syscall.StringToUTF16Ptr(commandLine),
		nil,
		nil,
		true,
		CREATE_SUSPENDED|CREATE_NO_WINDOW,
		nil,
		nil,
		startupInfo,
		procInfo)

	if errCreateProcess != nil && errCreateProcess.Error() != "The operation completed successfully." {
		log.Fatal(fmt.Sprintf("[!]Error calling CreateProcess:\r\n%s", errCreateProcess.Error()))
	}

	//Close the stdout and stderr write handles
	errCloseHandle := syscall.CloseHandle(stdOutWrite)
	if errCloseHandle != nil {
		output.VerbosePrint(fmt.Sprintf("[!]Error closing the STDOUT write handle:\r\n%s", errCloseHandle.Error()))
	}
	errCloseHandle = syscall.CloseHandle(stdErrWrite)
	if errCloseHandle != nil {
		output.VerbosePrint(fmt.Sprintf("[!]Error closing the STDERR write handle:\r\n%s", errCloseHandle.Error()))
	}

	return procInfo.Process, procInfo.ProcessId, stdOutRead, stdErrRead
}

func WaitReadBytes(handle syscall.Handle, tempBytes *[]byte, done *uint32) (err error) {

	var overlapped syscall.Overlapped

	var counter int
	var finished bool

	// Start reading from the pipe in another thread. That thread will block.
	go syncReadFile(handle, (uintptr)(unsafe.Pointer(&(*tempBytes)[0])), uintptr(len(*tempBytes)), done, &overlapped, &finished, &err)

	// Wait until ReadFile has stopped blocking or until timeout
	for finished == false && counter < 5000 {

		time.Sleep(50 * time.Millisecond)

		counter += 50
	}

	return err

}

func ReadFromPipes(stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte) (err error) {

	tempBytes := make([]byte, 8192)

	// Read STDOUT
	if stdout != 0 {

		for {
			var stdOutDone uint32

			err = WaitReadBytes(stdout, &tempBytes, &stdOutDone)

			if int(stdOutDone) == 0 {
				break
			}
			for _, b := range tempBytes {
				*stdoutBytes = append(*stdoutBytes, b)
			}

			if err != nil {

				if err.Error() != "The pipe has been ended." {
					break
				}

			}

		}
	}

	// Read STDERR
	if stderr != 0 {

		for {
			var stdErrDone uint32

			err = WaitReadBytes(stderr, &tempBytes, &stdErrDone)

			if int(stdErrDone) == 0 {
				break
			}
			for _, b := range tempBytes {
				*stderrBytes = append(*stderrBytes, b)
			}

			if err != nil {

				if err.Error() != "The pipe has been ended." {
					break
				}

			}

		}
	}

	return err
}
