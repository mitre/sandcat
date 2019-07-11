package modules

import (
	"os"
	"path/filepath"
	"fmt"
)

var cleanups []string
var payloads []string

// ApplyCleanup stores cleanup actions
func ApplyCleanup(command map[string]interface{}) {
	if command["cleanup"] != nil {
		cleanups = append(cleanups, command["cleanup"].(string))
	}
	if command["payload"] != nil {
		payloads = append(payloads, command["payload"].(string))
	}
}

// Cleanup runs all cleanup activities
func Cleanup(files string) {
	for i := range cleanups {
		cmd := Decode(cleanups[len(cleanups)-1-i])
		fmt.Println(fmt.Sprintf("[+] Cleanup: %s", cmd))
		Execute(string(cmd))
	}
	for _, value := range payloads {
		os.Remove(filepath.Join(files, value))
	}
	cleanups = cleanups[:0]
	payloads = payloads[:0]
}
