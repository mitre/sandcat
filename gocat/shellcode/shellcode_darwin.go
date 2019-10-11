package shellcode

import (
	"../output"
)

// Runner runner
func Runner(shellcode []byte) bool {
	output.VerbosePrint("[!] Shellcode executor for darwin not available")
	return false
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	return false
}
