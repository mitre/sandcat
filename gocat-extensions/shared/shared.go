// +build cgo

package main

import "C"

import (
    "strconv"
	"strings"

	"github.com/mitre/gocat/contact"
	"github.com/mitre/gocat/core"
)

var (
    key        = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"
	server     = "http://localhost:8888"
	paw        = ""
	group      = "red"
	c2Protocol = "HTTP"
	c2Key      = ""
	listenP2P  = "false" // need to set as string to allow ldflags -X build-time variable change on server-side.
    runOnInit  = "false" // need to set as string to allow ldflags -X build-time variable change on server-side.
	httpProxyGateway = ""
    running    = false
)

func init() {
    parsedRunOnInit, err := strconv.ParseBool(runOnInit)
	if err != nil {
		parsedRunOnInit = false
	}

    if parsedRunOnInit {
        VoidFunc()
    }
}

//export VoidFunc
func VoidFunc() {
    if running {
        return
    }

    running = true
    parsedListenP2P, err := strconv.ParseBool(listenP2P)
	if err != nil {
		parsedListenP2P = false
	}

	trimmedServer := strings.TrimRight(server, "/")
	contactConfig := map[string]string{
		"c2Name": c2Protocol,
		"c2Key": c2Key,
		"httpProxyGateway": httpProxyGateway,
	}
	tunnelConfig, err := contact.BuildTunnelConfig("", "", trimmedServer, "", "")
	if err != nil {
		return
	}
	core.Core(trimmedServer, tunnelConfig, group, 0, contactConfig, parsedListenP2P, false, paw, "")
}

func main() {}
