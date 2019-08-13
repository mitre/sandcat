package main

import (
	"crypto/tls"
	"fmt"
	"flag"
	"net/http"
	"os"
	"os/user"
	"time"
	"reflect"
	"./api"
	"./cleanup"
	"./util"
)

var iteration = 60

func runBeaconIteration(server string, paw string, group string, files string) {
	fmt.Println("[+] Beacon firing")
	commands := api.Beacon(server, paw, group, files)
	if commands != nil && len(commands.([]interface{})) > 0 {
		cmds := reflect.ValueOf(commands)
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i).Elem().String()
			fmt.Println("[+] Running instruction")
			command := util.Unpack([]byte(cmd))
			api.Drop(server, files, command)
			api.Results(server, paw, command)
			cleanup.Apply(command)
		}
	} else {
		cleanup.Run(files)
		time.Sleep(time.Duration(iteration) * time.Second)
	}
}

func runAutonomousIteration(server string, paw string, group string, files string) {
    fmt.Println("[+] Autonomous iteration running")
    time.Sleep(time.Duration(iteration) * time.Second)
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s$%s", host, user.Username)
	files := os.TempDir()

	server := flag.String("server", "http://localhost:8888", "The fqdn of CALDERA")
	group := flag.String("group", "my_group", "Attach a group to this agent")
	flag.Parse()

	for {
		runBeaconIteration(*server, paw, *group, files)
	}
}

var key = "OPU8GIV9Z7EIMNS5QPTN5X4DDSZ33U"