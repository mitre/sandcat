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

	"./api"
	"./execute"
	"./util"
)

func askForInstructions(profile map[string]interface{}) {
	beacon := api.Instructions(profile)
	if beacon["sleep"] != nil {
		profile["sleep"] = beacon["sleep"]
	}
	if beacon["instructions"] != nil && len(beacon["instructions"].([]interface{})) > 0 {
		cmds := reflect.ValueOf(beacon["instructions"])
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i).Elem().String()
			command := util.Unpack([]byte(cmd))
			go api.ExecuteInstruction(command, profile)
			util.Sleep(command["sleep"].(float64))
		}
	} else {
		util.Sleep(float64(profile["sleep"].(int)))
	}
}

func buildProfile(server string, group string, executors []string) map[string]interface{} {
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s$%s$%s", host, user.Username, key)
	profile := make(map[string]interface{})
	profile["paw"] = paw
	profile["server"] = server
	profile["group"] = group
	profile["architecture"] = runtime.GOARCH
	profile["platform"] = runtime.GOOS
	profile["location"] = os.Args[0]
	profile["sleep"] = 60
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

var key = "6JFMEPP68I7VHLL7I4COJOALZK4PG9"