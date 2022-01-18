package util

type NativeCmdResult struct {
	Stdout []byte
	Stderr []byte
	Err error
}

type NativeMethod func ([]string) NativeCmdResult

// Map command names to golang functions
var NativeMethods map[string]NativeMethod

func init() {
	NativeMethods = make(map[string]NativeMethod)
}