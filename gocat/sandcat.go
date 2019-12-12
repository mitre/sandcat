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
    defaultServer = "http://localhost:8888"
    defaultGroup = "my_group"
    defaultSleep = "60"
    c2Name = "HTTP"
    c2Key = ""
)

func main() {
	var executors execute.ExecutorFlags
	server := flag.String("server", defaultServer, "The FQDN of the server")
	group := flag.String("group", defaultGroup, "Attach a group to this agent")
	sleep := flag.String("sleep", defaultSleep, "Initial sleep value for sandcat (integer in seconds)")
	delay := flag.Int("delay", 0, "Delay starting this agent by n-seconds")
	verbose := flag.Bool("v", false, "Enable verbose output")
	c2 := flag.String("c2", c2Name, "C2 Channel for agent")

	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	
	c2Config := map[string]string{"c2Name": *c2, "c2Key": c2Key}

	core.Core(*server, *group, *sleep, *delay, executors, c2Config, *verbose)
}