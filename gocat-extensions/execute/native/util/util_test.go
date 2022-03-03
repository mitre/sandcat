package util

import (
	"errors"
	"testing"
)

func TestGenerateErrorResult(t *testing.T) {
	want := "test error msg"
	result := GenerateErrorResult(errors.New(want))
	verifyResult(t, result, "", want, want)
}

func TestGenerateErrorResultFromString(t *testing.T) {
	want := "test error msg"
	result := GenerateErrorResultFromString(want)
	verifyResult(t, result, "", want, want)
}

func verifyResult(t *testing.T, result NativeCmdResult, expectedStdout, expectedStderr, expectedErrMsg string) {
	if string(result.Stdout) != expectedStdout {
		t.Errorf("Expected stdout of '%s', got: %s", expectedStdout, string(result.Stdout))
	}
	if string(result.Stderr) != expectedStderr {
		t.Errorf("Expected stderr of '%s', got: %s", expectedStderr, string(result.Stderr))
	}
	if len(expectedErrMsg) > 0 {
		if result.Err == nil {
			t.Errorf("Expected error, received none.")
		} else if result.Err.Error() != expectedErrMsg {
			t.Errorf("Expected error message '%s', got: %s", expectedErrMsg, result.Err.Error())
		}
	} else if result.Err != nil {
		t.Errorf("Expected no error, got %v", result.Err)
	}
}