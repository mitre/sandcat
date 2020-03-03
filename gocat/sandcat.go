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
	listenP2P = false
)

func main() {
	var executors util.ListFlags
	server := flag.String("server", server, "The FQDN of the server")
	group := flag.String("group", "red", "Attach a group to this agent")
	c2 := flag.String("c2", c2Name, "C2 Channel for agent")
	delay := flag.Int("delay", 0, "Delay starting this agent by n-seconds")
	verbose := flag.Bool("v", false, "Enable verbose output")
	listenP2P := flag.Bool("listenP2p", listenP2P, "Enable peer-to-peer receivers")

	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	
	c2Config := map[string]string{"c2Name": *c2, "c2Key": c2Key}
	core.Core(*server, *group, *delay, executors, c2Config, *listenP2P, *verbose)
}