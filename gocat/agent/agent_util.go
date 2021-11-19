package agent

import (
	"os"
	"os/user"
	"os/exec"
	"time"
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

// Creates payload from []bytes
func writePayloadBytes(location string, payload []byte) error {
	dst, err := os.Create(location)
	if err != nil {
		return err
	} else {
		defer dst.Close()
		if _, err = dst.Write(payload); err != nil {
			return err
		} else if err = os.Chmod(location, 0700); err != nil {
			return err
		} else {
			return nil
		}
	}
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

func getFormattedTimestamp(timestamp time.Time, dateFormat string) (string) {
    return timestamp.Format(dateFormat)
}
