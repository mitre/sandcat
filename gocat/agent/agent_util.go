package agent

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/mitre/gocat/output"
)

// Checks for a file
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func getUsername() (string, error) {
	if userInfo, err := user.Current(); err != nil {
		if usernameBytes, err := exec.Command("whoami").CombinedOutput(); err == nil {
			return string(usernameBytes), nil
		} else {
			return "", err
		}
	} else {
		return userInfo.Username, nil
	}
}

func getFormattedTimestamp(timestamp time.Time, dateFormat string) string {
	return timestamp.Format(dateFormat)
}

func getExecutablePath() (string) {
	path, err := os.Executable()
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("Error obtaining executable path: %s", err.Error()))
		output.VerbosePrint("Obtaining path from command-line argument instead.")
		return os.Args[0]
	}
	return path
}