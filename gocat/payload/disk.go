package payload

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitre/gocat/output"
)

// Will download the specified payload data to disk using the specified filename.
// Returns filepath of the payload and any errors that occurred. If the payload already exists,
// no error will be returned.
func WriteToDisk(filename string, payloadBytes []byte) (string, error) {
	location := filepath.Join(filename)
	if !FileExists(location) {
		output.VerbosePrint(fmt.Sprintf("[*] Writing payload %s to disk at %s", filename, location))
		return location, WriteBytes(location, payloadBytes)
	}
	output.VerbosePrint(fmt.Sprintf("[*] File %s already exists", filename))
	return location, nil
}

// Creates from []bytes
func WriteBytes(location string, payload []byte) error {
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

//Determine if any payloads are not on disk
func CheckIfOnDisk(payloads []string) []string {
	var missing []string
	for i := range payloads {
		if !FileExists(filepath.Join(payloads[i])) {
			missing = append(missing, payloads[i])
		}
	}
	return missing
}

// Checks for a file
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}
