// +build windows

package donut

import (
	"fmt"
	"log"
	"syscall"

	"github.com/mitre/sandcat/gocat/output"
)

const (
	MEM_COMMIT             	= 0x1000
	MEM_RESERVE            	= 0x2000

	CREATE_SUSPENDED       	= 0x4
	CREATE_NO_WINDOW       	= 0x08000000

	SW_HIDE 				= 0
)

func CreateSuspendedProcessWithIORedirect(commandLine string) (syscall.Handle, syscall.Handle, syscall.Handle) {

	// Create anonymous pipe for STDOUT
	var stdOutRead syscall.Handle
	var stdOutWrite syscall.Handle

	errStdOutPipe := syscall.CreatePipe(&stdOutRead, &stdOutWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	errStdOutHandle := syscall.SetHandleInformation(stdOutRead, syscall.HANDLE_FLAG_INHERIT, 0)
	if errStdOutPipe != nil && errStdOutHandle != nil{
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
		CREATE_SUSPENDED | CREATE_NO_WINDOW,
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

	return procInfo.Process, stdOutRead, stdErrRead
}

func ReadFromPipes( stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte) (err error) {

	// Read STDOUT
	if stdout != 0	{
		var stdOutDone uint32
		var stdOutOverlapped syscall.Overlapped

		err = syscall.ReadFile(stdout, *stdoutBytes, &stdOutDone, &stdOutOverlapped)

		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!]Error reading the STDOUT pipe:\r\n%s", err.Error()))
		}
	}

	// Read STDERR
	if stderr != 0	{
		var stdErrDone uint32
		var stdErrOverlapped syscall.Overlapped

		err = syscall.ReadFile(stderr, *stderrBytes, &stdErrDone, &stdErrOverlapped)

		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!]Error reading the STDERR pipe:\r\n%s", err.Error()))
		}

		//Close the stdout and stderr read handles
		errCloseHandle := syscall.CloseHandle(stdout)
		if errCloseHandle != nil {
			output.VerbosePrint(fmt.Sprintf("[!]Error closing the STDOUT read handle:\r\n%s", errCloseHandle.Error()))
		}
		errCloseHandle = syscall.CloseHandle(stderr)
		if errCloseHandle != nil {
			output.VerbosePrint(fmt.Sprintf("[!]Error closing the STDERR read handle:\r\n%s", errCloseHandle.Error()))
		}
	}

	return
}