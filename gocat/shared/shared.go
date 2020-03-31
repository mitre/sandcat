// +build cgo

package main

import "C"
import (
	"strconv"

	"github.com/mitre/sandcat/gocat/core"
)

var (
	key       = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"
	server    = "http://localhost:8888"
	group     = "red"
	listenP2P = "false"
	c2Name    = "HTTP"
	c2Key     = ""
)

//export VoidFunc
func VoidFunc() {
	parsedListenP2P, _ := strconv.ParseBool(listenP2P)

	c2Config := map[string]string{"c2Name": c2Name, "c2Key": c2Key}
	core.Core(server, group, 0, nil, c2Config, parsedListenP2P, false)
}

func main() {}
