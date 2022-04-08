package core

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/mitre/gocat/agent"
	"github.com/mitre/gocat/contact"
	"github.com/mitre/gocat/output"

	_ "github.com/mitre/gocat/execute/donut"     // necessary to initialize all submodules
	_ "github.com/mitre/gocat/execute/native"    // necessary to initialize all submodules
	_ "github.com/mitre/gocat/execute/shellcode" // necessary to initialize all submodules
	_ "github.com/mitre/gocat/execute/shells"    // necessary to initialize all submodules
)

// Initializes and returns sandcat agent.
func initializeCore(server string, tunnelConfig *contact.TunnelConfig, group string, contactConfig map[string]string, p2pReceiversOn bool, initialDelay int, verbose bool, paw string, originLinkID string) (*agent.Agent, error) {
	output.SetVerbose(verbose)
	output.VerbosePrint("Starting sandcat in verbose mode.")
	return agent.AgentFactory(server, tunnelConfig, group, contactConfig, p2pReceiversOn, initialDelay, paw, originLinkID)
}

//Core is the main function as wrapped by sandcat.go
func Core(server string, tunnelConfig *contact.TunnelConfig, group string, delay int, contactConfig map[string]string, p2pReceiversOn bool, verbose bool, paw string, originLinkID string) {
	sandcatAgent, err := initializeCore(server, tunnelConfig, group, contactConfig, p2pReceiversOn, delay, verbose, paw, originLinkID)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error when initializing agent: %s", err.Error()))
		output.VerbosePrint("[-] Exiting.")
	} else {
		sandcatAgent.Display()
		runAgent(sandcatAgent, contactConfig)
		sandcatAgent.Terminate()
	}
}

func runAgent(sandcatAgent *agent.Agent, c2Config map[string]string) {
	selectedContact, ok := contact.CommunicationChannels[c2Config["c2Name"]]
	if !ok {
		output.VerbosePrint(fmt.Sprintf("[!] Requested C2 config not available: %s", c2Config["c2Name"]))
		output.VerbosePrint("[-] Exiting.")
	} else {
		if selectedContact.SupportsContinuous() {
			runAgentContinuousMode(sandcatAgent, c2Config)
		} else {
			runAgentBeaconMode(sandcatAgent, c2Config)
		}
	}
}

func runAgentContinuousMode(sandcatAgent *agent.Agent, c2Config map[string]string) {
	// Comms should already be set up through C2Requirements Met
	// Messages are already being accumulated, and outgoing messages are ready to be sent out
	// What we need to do is to pull one server message at a time using GetBeaconBytes and process it

	lastDiscovery := time.Now()

	for {
		// Continuous beacon doesn't actually ping the server, just grabs the earliest unprocessed server message
		// Process server message
		serverMessage := sandcatAgent.Beacon()
		// Process server message
		if len(serverMessage) == 0 {
			// No instruction given from server, we just continue
			continue
		}
		sandcatAgent.SetPaw(serverMessage["paw"].(string))

		// Check if we need to change contacts
		checkAndHandleContactChange(sandcatAgent, serverMessage, c2Config)

		// Check if we need to update executors
		checkAndHandleExecutorChange(sandcatAgent, serverMessage)

		// Handle instructions
		checkAndHandleInstructions(sandcatAgent, serverMessage)

		// randomly check for dynamically discoverable peer agents on the network
		if findPeers(lastDiscovery, sandcatAgent) {
			lastDiscovery = time.Now()
		}
	}
}

// Establish contact with C2 and run instructions.
func runAgentBeaconMode(sandcatAgent *agent.Agent, c2Config map[string]string) {
	// Start main execution loop.
	watchdog := 0
	checkin := time.Now()
	lastDiscovery := time.Now()
	var sleepDuration float64

	for evaluateWatchdog(checkin, watchdog) {
		// Send beacon and get response.
		beacon := sandcatAgent.Beacon()

		// Process beacon response.
		if len(beacon) != 0 {
			sandcatAgent.SetPaw(beacon["paw"].(string))
			checkin = time.Now()
			sleepDuration = float64(beacon["sleep"].(int))
			watchdog = beacon["watchdog"].(int)
		} else {
			// Failed beacon
			if err := sandcatAgent.HandleBeaconFailure(); err != nil {
				output.VerbosePrint(fmt.Sprintf("[!] Error handling failed beacon: %s", err.Error()))
				return
			}
			sleepDuration = float64(15)
		}

		// Check if we need to change contacts
		checkAndHandleContactChange(sandcatAgent, beacon, c2Config)

		// Check if we need to update executors
		checkAndHandleExecutorChange(sandcatAgent, beacon)

		// Handle instructions
		checkAndHandleInstructions(sandcatAgent, beacon)

		// randomly check for dynamically discoverable peer agents on the network
		if findPeers(lastDiscovery, sandcatAgent) {
			lastDiscovery = time.Now()
		}

		sandcatAgent.Sleep(sleepDuration)
	}
}

// Returns true if agent should keep running, false if not.
func evaluateWatchdog(lastcheckin time.Time, watchdog int) bool {
	return watchdog <= 0 || float64(time.Now().Sub(lastcheckin).Seconds()) <= float64(watchdog)
}

func findPeers(last time.Time, sandcatAgent *agent.Agent) bool {
	minDiscoveryInterval := 300
	diff := float64(time.Now().Sub(last).Seconds())
	if diff >= float64(rand.Intn(120)+minDiscoveryInterval) {
		sandcatAgent.DiscoverPeers()
		return true
	} else {
		return false
	}
}

func checkAndHandleContactChange(sandcatAgent *agent.Agent, beacon map[string]interface{}, c2Config map[string]string) {
	if beacon["new_contact"] != nil {
		newChannel := beacon["new_contact"].(string)
		c2Config["c2Name"] = newChannel
		output.VerbosePrint(fmt.Sprintf("Received request to switch from C2 channel %s to %s", sandcatAgent.GetCurrentContactName(), newChannel))
		if err := sandcatAgent.AttemptSelectComChannel(c2Config, newChannel); err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error switching communication channels: %s", err.Error()))
		}
	}
}

func checkAndHandleExecutorChange(sandcatAgent *agent.Agent, beacon map[string]interface{}) {
	if beacon["executor_change"] != nil {
		if err := sandcatAgent.ProcessExecutorChange(beacon["executor_change"]); err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error updating executor: %s", err.Error()))
		}
	}
}

func checkAndHandleInstructions(sandcatAgent *agent.Agent, beacon map[string]interface{}) {
	if beacon["instructions"] != nil && len(beacon["instructions"].([]interface{})) > 0 {
		// Run commands and send results.
		instructions := reflect.ValueOf(beacon["instructions"])
		for i := 0; i < instructions.Len(); i++ {
			marshaledInstruction := instructions.Index(i).Elem().String()
			var instruction map[string]interface{}
			if err := json.Unmarshal([]byte(marshaledInstruction), &instruction); err != nil {
				output.VerbosePrint(fmt.Sprintf("[-] Error unpacking command: %v", err.Error()))
			} else {
				// If instruction is deadman, save it for later. Otherwise, run the instruction.
				if instruction["deadman"].(bool) {
					output.VerbosePrint(fmt.Sprintf("[*] Received deadman instruction %s", instruction["id"]))
					sandcatAgent.StoreDeadmanInstruction(instruction)
				} else {
					output.VerbosePrint(fmt.Sprintf("[*] Running instruction %s", instruction["id"]))
					go sandcatAgent.RunInstruction(instruction, true)
					sandcatAgent.Sleep(instruction["sleep"].(float64))
				}
			}
		}
	}
}
