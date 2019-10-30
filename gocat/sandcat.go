package main

import (
	"flag"

	"./core"
	"./execute"
)

func main() {
	var executors execute.ExecutorFlags
	server := flag.String("server", core.DefaultServer, "The FQDN of the server")
	group := flag.String("group", core.DefaultGroup, "Attach a group to this agent")
	sleep := flag.String("sleep", core.DefaultSleep, "Initial sleep value for sandcat (integer in seconds)")
	verbose := flag.Bool("v", false, "Enable verbose output")
	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	core.Core(*server, *group, *sleep, executors, *verbose)
}