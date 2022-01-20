package discovery

import (
	"errors"
	"fmt"
	"os"

	"github.com/mitre/gocat/execute/native/util"
)

func init() {
	util.NativeMethods["pwd"] = GetWorkingDirectory
}

// Returns current working directory. Ignores any provided args.
func GetWorkingDirectory(args []string) util.NativeCmdResult {
	var resultErr error
	var stderr string
	var stdout string
	workingDir, err := os.Getwd()
	if err != nil {
		stderr = fmt.Sprintf("Error finding current working directory: %s", err.Error())
		resultErr = errors.New(stderr)
	} else {
		stdout = fmt.Sprintf("Current working directory: %s", workingDir)
	}
	return util.NativeCmdResult{
		Stdout: []byte(stdout),
		Stderr: []byte(stderr),
		Err: resultErr,
	}
}
