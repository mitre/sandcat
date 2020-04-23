// +build cgo

package main

import "C"
import (
	"github.com/mitre/gocat/core"
)

var (
	key       = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"
	server    = "http://localhost:8888"
	paw       = ""
	group     = "red"
	listenP2P = false
	c2Name    = "HTTP"
	c2Key     = ""
)

//export VoidFunc
func VoidFunc() {
	c2Config := map[string]string{"c2Name": c2Name, "c2Key": c2Key}
	core.Core(server, group, 0, c2Config, listenP2P, false, paw)
}

func main() {}
