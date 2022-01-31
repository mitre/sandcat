package shells

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mitre/gocat/execute"
)

const MOCK_CWD = "/mock/working/dir"
const PROC_NAME = "proc"
const DUMMY_ERROR_MSG = "dummy error msg"
const TEST_TIMEOUT = 10
var TEST_TIME = time.Date(2000, time.November, 10, 15, 0, 0, 0, time.UTC)

func MockGetCwd() (string, error) {
	return MOCK_CWD, nil
}

func MockGetWindowsOsName() string {
	return "windows"
}

func MockGetLinuxOsName() string {
	return "linux"
}

func MockGetDummyOsName() string {
	return "DummyOS"
}

func MockPidGetter() int {
	return 123
}

func MockFileDeleter(file string) error {
	return nil
}

func MockFileDeleteFail(file string) error {
	return errors.New(DUMMY_ERROR_MSG)
}

func MockTimeGenerator() time.Time {
	return TEST_TIME
}

func DummyInstructionInfo() execute.InstructionInfo {
	return execute.InstructionInfo{
		Profile: nil,
		Instruction: nil,
		OnDiskPayloads: nil,
		InMemoryPayloads: nil,
	}
}

func compareFunctionAddr(t *testing.T, outputFunc interface{}, wantFunc interface{}) {
	outputFuncAddr := fmt.Sprintf("%v", outputFunc)
	wantFuncAddr := fmt.Sprintf("%v", wantFunc)
	if outputFuncAddr != wantFuncAddr {
		t.Errorf("got '%s' func address; expected '%s'", outputFuncAddr, wantFuncAddr)
	}
}

func generateProcExecutor(osGetter OsGetter) *Proc {
	procFuncHandles := &ProcFunctionHandles{
		cwdGetter: MockGetCwd,
		osGetter: osGetter,
		pidGetter: MockPidGetter,
		fileDeleter: MockFileDeleter,
		timeStampGenerator: MockTimeGenerator,
	}
	return GenerateProcExecutor(procFuncHandles)
}

func TestGenerateProcExecutor(t *testing.T) {
	want := &Proc{
		currDir: MOCK_CWD,
		name: PROC_NAME,
		osName: "DummyOS",
		pidStr: "123",
		fileDeleter: MockFileDeleter,
		timeStampGenerator: MockTimeGenerator,
	}
	generated := generateProcExecutor(MockGetDummyOsName)
	if generated.currDir != want.currDir {
		t.Errorf("got '%s' as executor's current dir; expected '%s'", generated.currDir, want.currDir)
	}
	if generated.name != want.name {
		t.Errorf("got '%s' as executor's name; expected '%s'", generated.name, want.name)
	}
	if generated.osName != want.osName {
		t.Errorf("got '%s' as executor's OS; expected '%s'", generated.osName, want.osName)
	}
	if generated.pidStr != want.pidStr {
		t.Errorf("got '%s' as executor's process ID string; expected '%s'", generated.pidStr, want.pidStr)
	}
	compareFunctionAddr(t, generated.fileDeleter, want.fileDeleter)
	compareFunctionAddr(t, generated.timeStampGenerator, want.timeStampGenerator)
}

func TestProcString(t *testing.T) {
	p := generateProcExecutor(MockGetDummyOsName)
	want := PROC_NAME
	output := p.String()
	if output != want {
		t.Errorf("got '%s'; expected '%s'", output, want)
	}
}

func TestProcCheckIfAvailable(t *testing.T) {
	p := generateProcExecutor(MockGetDummyOsName)
	want := true
	output := p.CheckIfAvailable()
	if output != want {
		t.Errorf("got '%t'; expected '%t'", output, want)
	}
}

func TestProcDownloadPayloadToMemory(t *testing.T) {
	p := generateProcExecutor(MockGetDummyOsName)
	want := false
	output := p.DownloadPayloadToMemory("dummy")
	if output != want {
		t.Errorf("got '%t'; expected '%t'", output, want)
	}
}

func TestGetExeAndArgsWindowsNoArgs(t *testing.T) {
	p := generateProcExecutor(MockGetWindowsOsName)
	cmd := "C:\\Users\\Public\\executable.exe"
	wantCmd := cmd
	outputCmd, outputArgs, err := p.getExeAndArgs(cmd)
	if outputCmd != wantCmd {
		t.Errorf("got '%s'; expected '%s'", outputCmd, wantCmd)
	}
	if len(outputArgs) != 0 {
		t.Errorf("got nonempty args %v; expected empty args", outputArgs)
	}
	if err != nil {
		t.Errorf("got non-nil error with message '%s'; expected no error", err.Error())
	}
}

func TestGetExeAndArgsWindowsNoPath(t *testing.T) {
	p := generateProcExecutor(MockGetWindowsOsName)
	cmd := "executable.exe arg1 arg2 \"arg with space\" -45"
	wantCmd := "executable.exe"
	wantArgs := []string{"arg1", "arg2", "arg with space", "-45"}
	outputCmd, outputArgs, err := p.getExeAndArgs(cmd)
	if outputCmd != wantCmd {
		t.Errorf("got '%s'; expected '%s'", outputCmd, wantCmd)
	}
	if !reflect.DeepEqual(outputArgs, wantArgs) {
		t.Errorf("got '%v'; expected '%v'", outputArgs, wantArgs)
	}
	if err != nil {
		t.Errorf("got non-nil error with message '%s'; expected no error", err.Error())
	}
}

func TestGetExeAndArgsLinuxNoArgs(t *testing.T) {
	p := generateProcExecutor(MockGetLinuxOsName)
	cmd := "/bin/bash"
	wantCmd := cmd
	outputCmd, outputArgs, err := p.getExeAndArgs(cmd)
	if outputCmd != wantCmd {
		t.Errorf("got '%s'; expected '%s'", outputCmd, wantCmd)
	}
	if len(outputArgs) != 0 {
		t.Errorf("got nonempty args %v; expected empty args", outputArgs)
	}
	if err != nil {
		t.Errorf("got non-nil error with message '%s'; expected no error", err.Error())
	}
}

func TestGetExeAndArgsLinuxNoPath(t *testing.T) {
	p := generateProcExecutor(MockGetLinuxOsName)
	cmd := "binary -c arg1 arg2 \"arg with space\" -45 > file.txt"
	wantCmd := "binary"
	wantArgs := []string{"-c", "arg1", "arg2", "arg with space", "-45", ">", "file.txt"}
	outputCmd, outputArgs, err := p.getExeAndArgs(cmd)
	if outputCmd != wantCmd {
		t.Errorf("got '%s'; expected '%s'", outputCmd, wantCmd)
	}
	if !reflect.DeepEqual(outputArgs, wantArgs) {
		t.Errorf("got '%v'; expected '%v'", outputArgs, wantArgs)
	}
	if err != nil {
		t.Errorf("got non-nil error with message '%s'; expected no error", err.Error())
	}
}

func testAndValidateCmd(t *testing.T, p *Proc, cmd, wantMsg, wantStatus, wantPid string, wantTimestamp time.Time) {
	outputMsgBytes, outputStatus, outputPid, outputTimestamp := p.Run(cmd, TEST_TIMEOUT, DummyInstructionInfo())
	outputMsg := string(outputMsgBytes)
	if outputMsg != wantMsg {
		t.Errorf("got '%s'; expected '%s'", outputMsg, wantMsg)
	}
	if outputStatus != wantStatus {
		t.Errorf("got '%s'; expected '%s'", outputStatus, wantStatus)
	}
	if outputPid != wantPid {
		t.Errorf("got '%s'; expected '%s'", outputPid, wantPid)
	}
	if !outputTimestamp.Equal(wantTimestamp) {
		t.Errorf("got '%s'; expected '%s'", outputTimestamp.String(), wantTimestamp.String())
	}
}

func TestDeleteSingleFileWindows(t *testing.T) {
	p := generateProcExecutor(MockGetWindowsOsName)
	cmd := "del C:\\path\\to\\file"
	wantMsg := "Removed file C:\\path\\to\\file."
	wantStatus := execute.SUCCESS_STATUS
	wantPid := "123"
	wantTimestamp := TEST_TIME
	testAndValidateCmd(t, p, cmd, wantMsg, wantStatus, wantPid, wantTimestamp)
}

func TestDeleteMultipleFileWindows(t *testing.T) {
	p := generateProcExecutor(MockGetWindowsOsName)
	cmd := "del C:\\path\\to\\file1 .\\file2 file3.txt"
	wantMsg := "Removed file C:\\path\\to\\file1.\nRemoved file .\\file2.\nRemoved file file3.txt."
	wantStatus := execute.SUCCESS_STATUS
	wantPid := "123"
	wantTimestamp := TEST_TIME
	testAndValidateCmd(t, p, cmd, wantMsg, wantStatus, wantPid, wantTimestamp)
}

func TestDeleteMultipleFilesFailureWindows(t *testing.T) {
	p := generateProcExecutor(MockGetWindowsOsName)
	p.fileDeleter = MockFileDeleteFail
	cmd := "del C:\\path\\to\\file1 .\\file2 file3.txt"
	wantMsg := fmt.Sprintf("Failed to remove C:\\path\\to\\file1: %s\nFailed to remove .\\file2: %s\nFailed to remove file3.txt: %s",
						   DUMMY_ERROR_MSG, DUMMY_ERROR_MSG, DUMMY_ERROR_MSG)
	wantStatus := execute.ERROR_STATUS
	wantPid := "123"
	wantTimestamp := TEST_TIME
	testAndValidateCmd(t, p, cmd, wantMsg, wantStatus, wantPid, wantTimestamp)
}

func TestDeleteSingleFileLinux(t *testing.T) {
	p := generateProcExecutor(MockGetLinuxOsName)
	cmd := "del /path/to/file"
	wantMsg := "Removed file /path/to/file."
	wantStatus := execute.SUCCESS_STATUS
	wantPid := "123"
	wantTimestamp := TEST_TIME
	testAndValidateCmd(t, p, cmd, wantMsg, wantStatus, wantPid, wantTimestamp)
}

func TestDeleteMultipleFileLinux(t *testing.T) {
	p := generateProcExecutor(MockGetLinuxOsName)
	cmd := "del /path/to/file1 ./file2 file3.txt"
	wantMsg := "Removed file /path/to/file1.\nRemoved file ./file2.\nRemoved file file3.txt."
	wantStatus := execute.SUCCESS_STATUS
	wantPid := "123"
	wantTimestamp := TEST_TIME
	testAndValidateCmd(t, p, cmd, wantMsg, wantStatus, wantPid, wantTimestamp)
}

func TestDeleteMultipleFilesFailureLinux(t *testing.T) {
	p := generateProcExecutor(MockGetLinuxOsName)
	p.fileDeleter = MockFileDeleteFail
	cmd := "del /path/to/file1 ./file2 file3.txt"
	wantMsg := fmt.Sprintf("Failed to remove /path/to/file1: %s\nFailed to remove ./file2: %s\nFailed to remove file3.txt: %s",
						   DUMMY_ERROR_MSG, DUMMY_ERROR_MSG, DUMMY_ERROR_MSG)
	wantStatus := execute.ERROR_STATUS
	wantPid := "123"
	wantTimestamp := TEST_TIME
	testAndValidateCmd(t, p, cmd, wantMsg, wantStatus, wantPid, wantTimestamp)
}
