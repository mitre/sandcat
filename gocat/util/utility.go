package util

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Encode base64 encodes bytes
func Encode(b []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(b))
}

// Decode base64 decodes a string
func Decode(s string) []byte {
	raw, _ := base64.StdEncoding.DecodeString(s)
	return raw
}

// Unpack converts bytes into JSON
func Unpack(b []byte) (out map[string]interface{}) {
	_ = json.Unmarshal(b, &out)
	return
}

// Exists checks for a file
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// CheckErrorMessage Check for error message
func CheckErrorMessage(err error) bool {
	if err != nil && err.Error() != "The operation completed successfully." {
		println(err.Error())
		return true
	}
	return false
}

// StringToByteArrayString transforms an input string to a byte string
func StringToByteArrayString(input string) ([]byte, error) {
	temp := removeWhiteSpace(input)
	temp = strings.Replace(temp, "0x", "", -1)
	temp = strings.Replace(temp, "\\x", "", -1)
	temp = strings.Replace(temp, ",", "", -1)
	return hex.DecodeString(temp)
}

// Sleep sleeps for a desired interval
func Sleep(interval float64) {
	time.Sleep(time.Duration(interval) * time.Second)
}

//WritePayload creates a payload on disk
func WritePayload(location string, resp *http.Response) {
	dst, _ := os.Create(location)
	defer dst.Close()
	_, _ = io.Copy(dst, resp.Body)
	os.Chmod(location, 0700)
}

//WritePayloadBytes creates payload from []bytes
func WritePayloadBytes(location string, payload []byte) {
	dst, _ := os.Create(location)
	defer dst.Close()
	_, _ = dst.Write(payload)
	os.Chmod(location, 0700)
}

//CheckPayloadsAvailable determines if any payloads are not on disk
func CheckPayloadsAvailable(payloads []string) []string {
	var missing []string
	for i := range payloads {
		if Exists(filepath.Join(payloads[i])) == false {
			missing = append(missing, payloads[i])
		}
	}
	return missing
}

//StopProcess kills a PID
func StopProcess(pid int) {
	proc, _ := os.FindProcess(pid)
	_ = proc.Kill()
}

func removeWhiteSpace(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, input)
}

func EvaluateWatchdog(lastcheckin time.Time, watchdog int) {
	if watchdog > 0 && float64(time.Now().Sub(lastcheckin).Seconds()) > float64(watchdog) {
		StopProcess(os.Getpid())
	}
}

type ListFlags []string

func (l *ListFlags) String() string {
	return fmt.Sprint(*l)
}

func (l *ListFlags) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		*l = append(*l, item)
	}
	return nil
}
