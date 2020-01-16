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
    defaultP2pReceiver = ""
    defaultP2pReceiverType = ""
)

func main() {
	var executors execute.ExecutorFlags
	server := flag.String("server", server, "The FQDN of the server")
	group := flag.String("group", "red", "Attach a group to this agent")
	c2 := flag.String("c2", c2Name, "C2 Channel for agent")
	delay := flag.Int("delay", 0, "Delay starting this agent by n-seconds")
	verbose := flag.Bool("v", false, "Enable verbose output")
	p2pReceiver := flag.String("p2pReceiver", defaultP2pReceiver, "Location to listen on for p2p forwarding.")
	p2pReceiverType := flag.String("p2pReceiverType", defaultP2pReceiverType, "P2P receiver method")

	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()

	c2Config := map[string]string{"c2Name": *c2, "c2Key": c2Key}
	p2pReceiverConfig := map[string]string{"p2pReceiver": *p2pReceiver, "p2pReceiverType": *p2pReceiverType}
	core.Core(*server, *group, *delay, executors, c2Config, p2pReceiverConfig, *verbose)
}