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
	PAGE_EXECUTE_READWRITE 	= 0x40

	CREATE_SUSPENDED       	= 0x4
	CREATE_NO_WINDOW       	= 0x08000000

	PROCESS_CREATE_THREAD 	= 0x2
	PROCESS_VM_OPERATION 	= 0x8
	PROCESS_VM_WRITE 		= 0x20
	PROCESS_VM_READ 		= 0x10
	SW_HIDE 				= 0
)

func CreateSuspendedProcessWIORedirect(commandLine string) (syscall.Handle, syscall.Handle, syscall.Handle) {

	// Create anonymous pipe for STDOUT
	var stdOutRead syscall.Handle
	var stdOutWrite syscall.Handle

	errStdOutPipe := syscall.CreatePipe(&stdOutRead, &stdOutWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	if errStdOutPipe != nil {
		output.VerbosePrint(fmt.Sprintf("[!]Error creating the STDOUT pipe:\r\n%s", errStdOutPipe.Error()))
	}

	// Create anonymous pipe for STDERR
	var stdErrRead syscall.Handle
	var stdErrWrite syscall.Handle

	errStdErrPipe := syscall.CreatePipe(&stdErrRead, &stdErrWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	if errStdErrPipe != nil {
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
		false,
		CREATE_SUSPENDED | CREATE_NO_WINDOW,
		nil,
		nil,
		startupInfo,
		procInfo)

	if errCreateProcess != nil && errCreateProcess.Error() != "The operation completed successfully." {
		log.Fatal(fmt.Sprintf("[!]Error calling CreateProcess:\r\n%s", errCreateProcess.Error()))
	}

	return procInfo.Process, stdOutRead, stdErrRead
}

func ReadFromPipes( stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte) (err error) {

	output.VerbosePrint("In ReadFromPipes")

	// Read STDOUT
	if stdout != 0	{
		var stdOutDone uint32
		var stdOutOverlapped syscall.Overlapped

		output.VerbosePrint("In stdout")

		//Try to call PeekNamedPipe
		syscall.FlushFileBuffers(stdout)

		err = syscall.ReadFile(stdout, *stdoutBytes, &stdOutDone, &stdOutOverlapped)

		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!]Error reading the STDOUT pipe:\r\n%s", err.Error()))
		}

		output.VerbosePrint("Finished stdout")
	}

	// Read STDERR
	if stderr != 0	{
		var stdErrDone uint32
		var stdErrOverlapped syscall.Overlapped

		output.VerbosePrint("In stderr")

		syscall.FlushFileBuffers(stderr)

		err = syscall.ReadFile(stderr, *stderrBytes, &stdErrDone, &stdErrOverlapped)

		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!]Error reading the STDOUT pipe:\r\n%s", err.Error()))
		}

		output.VerbosePrint("Finished stdout")
	}

	return
}