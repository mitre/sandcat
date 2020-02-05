package main

import (
	"flag"

	"./core"
	"./util"
)

/*
These default  values can be overridden during linking - server, group, and sleep can also be overridden
with command-line arguments at runtime.
*/
var (
    key = "JWHQZM9Z4HQOYICDHW4OCJAXPPNHBA"
    server = "http://localhost:8888"
    c2Name = "HTTP"
	c2Key = ""
)

func main() {
	var executors util.ListFlags
	var servers util.ListFlags

	flag.Var(&servers, "server", "Comma separated list of target servers (IP or FQDN)")
	c2 := flag.String("c2", c2Name, "C2 Channel for agent")
	delay := flag.Int("delay", 0, "Delay starting this agent by n-seconds")
	verbose := flag.Bool("v", false, "Enable verbose output")

	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	
	c2Config := map[string]string{"c2Name": *c2, "c2Key": c2Key}
	core.Core(servers, *delay, executors, c2Config, *verbose)
}