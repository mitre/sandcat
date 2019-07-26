package cleanup

import (
	"fmt"
	"../util"
	"../execute"
)

var cleanups []string

// Apply stores cleanup actions
func Apply(command map[string]interface{}) {
	if command["cleanup"] != nil && len(command["cleanup"].(string)) > 1 {
		cleanups = append(cleanups, command["cleanup"].(string))
	}
}

// Run executes all cleanup activities
func Run(files string) {
	for i := range cleanups {
		cmd := util.Decode(cleanups[len(cleanups)-1-i])
		fmt.Println(fmt.Sprintf("[+] Cleanup: %s", cmd))
		execute.Execute(string(cmd))
	}
	cleanups = cleanups[:0]
}
