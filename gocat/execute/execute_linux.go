package execute

import "syscall"

func getPlatformSysProcAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

