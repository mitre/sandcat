package privdetect

import (
    "syscall"
    "unsafe"
)
type Token uintptr

const (
	// do not reorder
	TokenUser = 1 + iota
	TokenGroups
	TokenPrivileges
	TokenOwner
	TokenPrimaryGroup
	TokenDefaultDacl
	TokenSource
	TokenType
	TokenImpersonationLevel
	TokenStatistics
	TokenRestrictedSids
	TokenSessionId
	TokenGroupsAndPrivileges
	TokenSessionReference
	TokenSandBoxInert
	TokenAuditPolicy
	TokenOrigin
	TokenElevationType
	TokenLinkedToken
	TokenElevation
	TokenHasRestrictions
	TokenAccessInformation
	TokenVirtualizationAllowed
	TokenVirtualizationEnabled
	TokenIntegrityLevel
	TokenUIAccess
	TokenMandatoryPolicy
	TokenLogonSid
	MaxTokenInfoClass
	errnoERROR_IO_PENDING = 997
)

var (
	modAdvapi32 = syscall.NewLazyDLL("Advapi32.dll")
	procGetTokenInformation = modAdvapi32.NewProc("GetTokenInformation")
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
)

func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	return e
}

func GetTokenInformation(token Token, infoClass uint32, info *byte, infoLen uint32, returnedLen *uint32) (err error) {
	r1, _, e1 := syscall.Syscall6(procGetTokenInformation.Addr(), 5, uintptr(token), uintptr(infoClass), uintptr(unsafe.Pointer(info)), uintptr(infoLen), uintptr(unsafe.Pointer(returnedLen)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func (token Token) IsElevated() bool {
	var isElevated uint32
	var outLen uint32
	err := GetTokenInformation(token, TokenElevation, (*byte)(unsafe.Pointer(&isElevated)), uint32(unsafe.Sizeof(isElevated)), &outLen)
	if err != nil {
		return false
	}
	return outLen == uint32(unsafe.Sizeof(isElevated)) && isElevated != 0
}

func Privlevel() string{
    token := Token(^uintptr(4 - 1))
    if token.IsElevated() ==true {
    	return "Elevated"
    } else {
    	return "User"
    }
}