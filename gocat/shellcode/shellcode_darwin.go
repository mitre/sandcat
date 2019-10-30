package shellcode

import (
	"../output"
)

// Runner runner
func Runner(shellcode []byte) (bool, string) {
	output.VerbosePrint("[!] Shellcode executor for darwin not available")
	return false, ERROR_PID
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	return false
}
