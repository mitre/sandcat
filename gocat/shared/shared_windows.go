package shared

import "C"
import (
	"../core"
	"../execute"
)

//export VoidFunc
func VoidFunc() {
	core.Core(core.DefaultServer, core.DefaultGroup, core.DefaultSleep, execute.ExecutorFlags{}, false)
}
