package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"../util"
	"../execute"
)

var cleanups []string
var payloads []string

// Apply stores cleanup actions
func Apply(command map[string]interface{}) {
	if command["cleanup"] != nil && len(command["cleanup"].(string)) > 1 {
		cleanups = append(cleanups, command["cleanup"].(string))
	}
	payload := command["payload"].(string)
	if len(payload) > 0 && !alreadyHave(payload) {
		payloads = append(payloads, payload)
	}
}

// Run executes all cleanup activities
func Run(profile map[string]string) {
	for i := range cleanups {
		cmd := util.Decode(cleanups[len(cleanups)-1-i])
		fmt.Println(fmt.Sprintf("[+] Cleanup: %s", cmd))
		execute.Execute(string(cmd), profile["executor"])
	}
	for _, value := range payloads {
		fmt.Println(fmt.Sprintf("[*] Removing payload: %s", value))
		os.Remove(filepath.Join(value))
	}
	cleanups = cleanups[:0]
	payloads = payloads[:0]
}

func alreadyHave(payload string) bool {
    for _, p := range payloads {
        if p == payload { return true }
    }
    return false
}