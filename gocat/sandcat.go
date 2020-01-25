package main

import (
	"flag"

	"./core"
	"./execute"
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
	apiBeacon = "/beacon"
	apiResult = "/result"
)

func main() {
	var executors execute.ExecutorFlags
	server := flag.String("server", server, "The FQDN of the server")
	c2 := flag.String("c2", c2Name, "C2 Channel for agent")
	delay := flag.Int("delay", 0, "Delay starting this agent by n-seconds")
	verbose := flag.Bool("v", false, "Enable verbose output")

	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	
	c2Config := map[string]string{"c2Name": *c2, "c2Key": c2Key, "apiBeacon": apiBeacon, "apiResult": apiResult}
	core.Core(*server, *delay, executors, c2Config, *verbose)
}