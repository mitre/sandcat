package discovery

import (
	"errors"
	"io/ioutil"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/mitre/gocat/execute/native"
)

func init() {
	native.NativeMethods["ListDir"] = ListDirectories
	native.NativeMethods["ls"] = ListDirectories
}

// Lists file information for each directory in the args list
func ListDirectories(dirList []string) native.NativeCmdResult {
	if len(dirList) == 0 {
		return handleSingleDir(".")
	}
	if len(dirList) == 1 {
		return handleSingleDir(dirList[0])
	}
	var stdoutLines []string
	var stderrLines []string
	var resultErr error
	var stderr string
	for _, dirName := range dirList {
		output, err := listDirectory(dirName)
		if err != nil {
			stderrLines = append(stderrLines, fmt.Sprintf("Error listing directory %s:", dirName))
			stderrLines = append(stderrLines, err.Error(), "")
		} else {
			stdoutLines = append(stdoutLines, fmt.Sprintf("Directory listing for %s:", dirName))
			stdoutLines = append(stdoutLines, output, "")
		}
	}
	if len(stderrLines) > 0 {
		stderr = strings.Join(stderrLines[:], "\n")
		resultErr = errors.New(stderr)
	}
	return native.NativeCmdResult{
		Stdout: []byte(strings.Join(stdoutLines[:], "\n")),
		Stderr: []byte(stderr),
		Err: resultErr,
	}
}

func listDirectory(dirName string) (string, error) {
	dirEntries, err := ioutil.ReadDir(dirName)
	if err != nil {
		return "", err
	}
	sizeWidth := getSizeWidth(dirEntries)
	var fileListing []string
	for _, fileInfo := range dirEntries {
		fileListing = append(fileListing, getFileEntryInfoStr(fileInfo, sizeWidth))
	}
	return strings.Join(fileListing[:], "\n"), nil
}

func getFileEntryInfoStr(fileInfo os.FileInfo, sizeWidth int) string {
	fileName := fileInfo.Name()
	if fileInfo.IsDir() {
		fileName += "/"
	}
	return fmt.Sprintf("%s  %*d  %s", fileInfo.Mode().String(), sizeWidth, fileInfo.Size(), fileName)
}

func handleSingleDir(dirName string) native.NativeCmdResult {
	output, err := listDirectory(dirName)
	if err != nil {
		return native.NativeCmdResult{
			Stdout: nil,
			Stderr: []byte(err.Error()),
			Err: err,
		}
	}
	return native.NativeCmdResult{
		Stdout: []byte(output),
		Stderr: nil,
		Err: nil,
	}
}

func getSizeWidth(dirEntries []os.FileInfo) int {
	maxSizeStrLen := 0
	sizeStrLen := 0
	for _, fileInfo := range dirEntries {
		sizeStrLen = getFileSizeStrLen(fileInfo.Size())
		if sizeStrLen > maxSizeStrLen {
			maxSizeStrLen = sizeStrLen
		}
	}
	return maxSizeStrLen
}

func getFileSizeStrLen(fileSize int64) int {
	return int(math.Log10(float64(fileSize))) + 1
}