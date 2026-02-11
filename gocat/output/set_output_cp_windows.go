// +build windows

package output

import (
    "errors"
    "fmt"

    "golang.org/x/sys/windows"
)

func SetConsoleOutputUTF8() error {
    err := windows.SetConsoleOutputCP(65001) // UTF-8 code page identifier
    if err != nil {
        return errors.New(fmt.Sprintf("Failed to set console output to UTF8: %s", err.Error()))
    }
    return nil
}

