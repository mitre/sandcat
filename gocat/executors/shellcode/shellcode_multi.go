// +build !windows !linux

package shellcode

// Runner runner
func Runner(shellcode []byte) (bool, string) {
	return false, "Runner not available"
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	return false
}