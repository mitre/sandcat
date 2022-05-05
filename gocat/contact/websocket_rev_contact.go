package contact

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"github.com/mitre/gocat/output"
)

var (
	websocket_url   = "/ws_interactive"
	websocket_proto = "ws"
	ws_userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"
)

//API communicates through HTTP
type Websocket struct {
	name             string
	client           *http.Client
	upstreamDestAddr string
	ws_client        *websocket.Conn
}

func init() {
	CommunicationChannels["Websocket"] = &Websocket{name: "Websocket"}
}

//GetInstructions sends a beacon and returns response.
func (a *Websocket) GetBeaconBytes(profile map[string]interface{}) []byte {
	output.VerbosePrint("[*] Getting commands")
	data, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot request beacon. Error with profile marshal: %s", err.Error()))
		return nil
	} else {
		// address := fmt.Sprintf("%s%s", a.upstreamDestAddr, apiBeacon)
		return a.request(data)
	}
}

// Return the file bytes for the requested payload.
func (a *Websocket) GetPayloadBytes(profile map[string]interface{}, payload string) ([]byte, string) {
	// Not implemented due to interactive nature.
	return nil, ""
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (a *Websocket) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
	upstreamurl, err := url.Parse(a.upstreamDestAddr)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("Invalid URL: %v", err))
		return false, nil
	}
	a.SetUpstreamDestAddr(fmt.Sprintf("%s://%s/%s", websocket_proto, upstreamurl.Host, websocket_url))
	// a.SetUpstreamDestAddr("ws://localhost:7012/ws_interactive")

	output.VerbosePrint(fmt.Sprintf("Interactive endpoint=%s", a.upstreamDestAddr))

	// Gorilla handles the HTTP upgrade to websocket so we don't need that client anymore.
	// Using a unique name ws_client to avoid name confliction with api.go
	c, _, err := websocket.DefaultDialer.Dial(a.upstreamDestAddr, nil)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("dial: %v", err))
		return false, nil
	}
	a.ws_client = c
	return true, nil
}

func (a *Websocket) SetUpstreamDestAddr(upstreamDestAddr string) {
	upstreamDestAddr = "ws://localhost:7013/ws_interactive"
	a.upstreamDestAddr = upstreamDestAddr
}

// SendExecutionResults will send the execution results to the upstream destination.
func (a *Websocket) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	output.VerbosePrint("[*] Sending results")
	_ = fmt.Sprintf("%s%s", a.upstreamDestAddr, apiBeacon)
	profileCopy := make(map[string]interface{})
	for k, v := range profile {
		profileCopy[k] = v
	}
	results := make([]map[string]interface{}, 1)
	results[0] = result
	profileCopy["results"] = results
	data, err := json.Marshal(profileCopy)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send results. Error with profile marshal: %s", err.Error()))
	} else {
		a.request(data)
	}
}

func (a *Websocket) GetName() string {
	return a.name
}

func (a *Websocket) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	// Not implemented due to interactive nature.
	return nil
}

func (a *Websocket) request(data []byte) []byte {
	output.VerbosePrint(string(data))
	encodedData := []byte(base64.StdEncoding.EncodeToString(data))
	output.VerbosePrint("[*] Making request")

	err := a.ws_client.WriteMessage(websocket.TextMessage, encodedData)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send websocket message: %s", err.Error()))
		return nil
	}
	_, message, err := a.ws_client.ReadMessage()
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot recieve websocket message: %s", err.Error()))
		return nil
	}

	decodedData, err := base64.StdEncoding.DecodeString(string(message))
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot decode websocket message: %s", err.Error()))
		return nil
	}
	output.VerbosePrint(fmt.Sprintf("[*] Decoded message:\n %s", decodedData))
	var jsonData interface{}
	err = json.Unmarshal(decodedData, &jsonData)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot unmarshal json data: %s", err.Error()))
		return nil
	}
	// if val, ok := jsonData["sleep"]; ok {
	// 	jsonData["sleep"] = float64(0)
	// }

	return decodedData
}
