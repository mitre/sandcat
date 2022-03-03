package util

import (
	"errors"
)

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

func GenerateErrorResult(err error) NativeCmdResult {
	return NativeCmdResult{
		Stdout: nil,
		Stderr: []byte(err.Error()),
		Err: err,
	}
}

func GenerateErrorResultFromString(errMsg string) NativeCmdResult {
	return NativeCmdResult{
		Stdout: nil,
		Stderr: []byte(errMsg),
		Err: errors.New(errMsg),
	}
}