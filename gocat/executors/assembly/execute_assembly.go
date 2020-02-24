package assembly

import (
	"../execute"
)

type ExecuteAssembly struct {
	shortName string
	path string
}

func init() {
	assembly := &ExecuteAssembly{
		shortName: "assembly",
		path: "execute-assembly",
	}
	if assembly.CheckIfAvailable() {
		execute.Executors[assembly.shortName] = assembly
	}
}

func (e *ExecuteAssembly) Run(command string, timeout int) ([]byte, string, string) {
	return runAssembly(command, "", timeout)
}

func (e *ExecuteAssembly) String() string {
	return e.shortName
}

func (e *ExecuteAssembly) CheckIfAvailable() bool {
	return checkIfAvailable()
}
