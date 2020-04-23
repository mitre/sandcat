// +build windows

package donut

import (
	"fmt"
	"log"
	"syscall"
	"time"
	"unsafe"

	"github.com/mitre/sandcat/gocat/output"
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

	output.VerbosePrint(fmt.Sprintf("DEBUG: in WaitReadBytes"))

	var counter int
	var finished bool

	output.VerbosePrint(fmt.Sprintf("DEBUG: starting ReadFile"))

	go syncReadFile(handle, (uintptr)(unsafe.Pointer(&(*tempBytes)[0])), uintptr(len(*tempBytes)), done, &overlapped, &finished, &err)

	for finished == false && counter < 5000 {

		output.VerbosePrint(fmt.Sprintf("[!]Status:%s\tCounter%s\r\n", finished, counter))

		time.Sleep(50 * time.Millisecond)

		counter += 50 // incremember by 50 milliseconds
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: timed out"))

	return err

}

func ReadFromPipes(stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte) (err error) {

	tempBytes := make([]byte, 1)

	output.VerbosePrint(fmt.Sprintf("DEBUG: in ReadFromPipes"))
	// Read STDOUT
	if stdout != 0 {

		for {
			var stdOutDone uint32

			output.VerbosePrint(fmt.Sprintf("DEBUG: reading STDOUT"))

			err = WaitReadBytes(stdout, &tempBytes, &stdOutDone)

			output.VerbosePrint(fmt.Sprintf("DEBUG: read STDOUT"))

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

				output.VerbosePrint(fmt.Sprintf("[!]Error reading the STDOUT pipe:\r\n%s", err.Error()))
			}

		}
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: done wih STDOUT"))

	// Read STDERR
	if stderr != 0 {

		for {
			var stdErrDone uint32

			output.VerbosePrint(fmt.Sprintf("DEBUG: reading STDERR"))

			err = WaitReadBytes(stderr, &tempBytes, &stdErrDone)

			output.VerbosePrint(fmt.Sprintf("DEBUG: read STDERR"))

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

				output.VerbosePrint(fmt.Sprintf("[!]Error reading the STDERR pipe:\r\n%s", err.Error()))
			}

		}
	}

	output.VerbosePrint(fmt.Sprintf("DEBUG: done wih STDERR"))

	output.VerbosePrint(fmt.Sprintf("DEBUG: Done with ReadFromPipes"))

	return err
}
