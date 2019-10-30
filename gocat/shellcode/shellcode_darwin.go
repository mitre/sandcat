package shellcode

import (
	"../output"
)

// Runner runner
func Runner(shellcode []byte) (bool, int) {
	output.VerbosePrint("[!] Shellcode executor for darwin not available")
	return false, 1
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	return false
}
