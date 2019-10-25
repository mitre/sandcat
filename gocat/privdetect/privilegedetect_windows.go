package privdetect

import (
        "golang.org/x/sys/windows"
)

func Privlevel() string{
    token := windows.GetCurrentProcessToken()
    
    if token.IsElevated() ==true {
    	return "Elevated"
    } else {
    	return "User"
    }
}