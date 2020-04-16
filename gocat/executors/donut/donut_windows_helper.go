// +build windows

package donut

import (
	"fmt"
	"log"
	"syscall"
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

func CreateSuspendedProcessWIORedirect(commandLine string) (syscall.Handle, syscall.Handle, syscall.Handle, syscall.Handle) {

	// Create anonymous pipe for STDIN
	var stdInRead syscall.Handle
	var stdInWrite syscall.Handle

	errStdInPipe := syscall.CreatePipe(&stdInRead, &stdInWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	if errStdInPipe != nil {
		log.Fatal(fmt.Sprintf("[!]Error creating the STDIN pipe:\r\n%s", errStdInPipe.Error()))
	}

	// Create anonymous pipe for STDOUT
	var stdOutRead syscall.Handle
	var stdOutWrite syscall.Handle

	errStdOutPipe := syscall.CreatePipe(&stdOutRead, &stdOutWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	if errStdOutPipe != nil {
		log.Fatal(fmt.Sprintf("[!]Error creating the STDOUT pipe:\r\n%s", errStdOutPipe.Error()))
	}

	// Create anonymous pipe for STDERR
	var stdErrRead syscall.Handle
	var stdErrWrite syscall.Handle

	errStdErrPipe := syscall.CreatePipe(&stdErrRead, &stdErrWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	if errStdErrPipe != nil {
		log.Fatal(fmt.Sprintf("[!]Error creating the STDERR pipe:\r\n%s", errStdErrPipe.Error()))
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

	return procInfo.Process, stdInWrite, stdOutRead, stdErrRead
}