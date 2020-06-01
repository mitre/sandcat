package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"io/ioutil"
	"sync"
	"strconv"
	"time"

	"github.com/mitre/gocat/output"
	"github.com/mitre/gocat/contact"
)

var (
	beaconEndpoint = "/beacon"
	payloadEndpoint = "/file/download"
	httpProxyName = "HTTP"
	maxPortRetries = 5
	minReceiverPort = 50000
	maxReceiverPort = 63000
)

//HttpReceiver forwards data received from HTTP requests to the upstream server via HTTP. Implements the P2pReceiver interface.
type HttpReceiver struct {
	upstreamServer string
	port int
	bindPortStr string
	receiverName string
	upstreamComs contact.Contact
	httpServer *http.Server
	waitgroup *sync.WaitGroup
	receiverContext context.Context
	receiverCancelFunc context.CancelFunc
	urlList []string // list of HTTP urls that external machines can use to reach this receiver.
}

func init() {
	P2pReceiverChannels[httpProxyName] = &HttpReceiver{}
}

func (h *HttpReceiver) InitializeReceiver(server string, upstreamComs contact.Contact, waitgroup *sync.WaitGroup) error {
	err := h.initializeReceiverPort()
	if err != nil {
		return err
	}
	h.upstreamServer = server
	h.receiverName = httpProxyName
	h.upstreamComs = upstreamComs
	h.httpServer = &http.Server{
		Addr: h.bindPortStr,
		Handler: nil,
	}
	h.urlList, err = h.getReachableUrls()
	if err != nil {
		return err
	}
	h.waitgroup = waitgroup
	h.receiverContext, h.receiverCancelFunc = context.WithTimeout(context.Background(), 5*time.Second)
	return nil
}

func (h *HttpReceiver) RunReceiver() {
	output.VerbosePrint(fmt.Sprintf("[*] Starting HTTP proxy receiver on local port %d", h.port))
	output.VerbosePrint(fmt.Sprintf("[*] HTTP proxy receiver has upstream contact at %s", h.upstreamServer))
	h.startHttpProxy()
}

func (h *HttpReceiver) Terminate() {
	defer func() {
		h.waitgroup.Done()
		h.receiverCancelFunc()
	}()
	if err := h.httpServer.Shutdown(h.receiverContext); err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error when shutting down HTTP receiver server: %s", err.Error()))
	}
}

func (h *HttpReceiver) UpdateUpstreamServer(newServer string) {
	h.upstreamServer = newServer
}

func (h *HttpReceiver) UpdateUpstreamComs(newComs contact.Contact) {
	h.upstreamComs = newComs
}

func (h *HttpReceiver) GetReceiverAddresses() []string {
	return h.urlList
}

// Helper method for StartReceiver. Starts HTTP proxy to forward messages from peers to the C2 server.
func (h *HttpReceiver) startHttpProxy() {
	http.HandleFunc(beaconEndpoint, h.handleBeaconEndpoint)
	http.HandleFunc(payloadEndpoint, h.handlePayloadEndpoint)
	if err := http.ListenAndServe(h.bindPortStr, nil); err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] HTTP proxy error: %s", err.Error()))
	}
}

// Handle beacon/execution results sent to /beacon
func (h *HttpReceiver) handleBeaconEndpoint(writer http.ResponseWriter, reader *http.Request) {
	// Get data from the message that client peer sent.
	body, err := ioutil.ReadAll(reader.Body)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error: could not read data from beacon request: %s", err.Error()))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	reader.Body = ioutil.NopCloser(bytes.NewReader(body))

	// Extract profile from the data.
	profileData, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error: malformed profile base64 received: %s", err.Error()))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	profile := make(map[string]interface{})
	if err = json.Unmarshal(profileData, &profile); err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error: malformed profile data received on beacon endpoint: %s", err.Error()))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make sure we forward the request to the right place.
	profile["server"] = h.upstreamServer

	// Check if profile contains execution results
	if results, ok := profile["results"]; ok {
		output.VerbosePrint("[*] HTTP proxy: handling execution results from client.")
		resultList := results.([]interface{})
		if len(resultList) > 0 {
			h.upstreamComs.SendExecutionResults(profile, resultList[0].(map[string]interface{}))
		} else {
			output.VerbosePrint("[!] Error: client sent empty result list.")
			http.Error(writer, "Empty result list received from client", http.StatusInternalServerError)
		}
	} else {
		output.VerbosePrint("[*] HTTP proxy: handling beacon request from client.")
		beaconResponse := h.upstreamComs.GetBeaconBytes(profile)
		encodedResponse := []byte(base64.StdEncoding.EncodeToString(beaconResponse))
		if err = sendResponseToClient(encodedResponse, nil, writer); err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error sending response to client: %s", err.Error()))
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Handle payload requests sent to /file/download
func (h *HttpReceiver) handlePayloadEndpoint(writer http.ResponseWriter, reader *http.Request) {
	output.VerbosePrint("[*] HTTP proxy: handling payload request from client.")

	// Get filename, paw, and platform from headers
	filenameReqHeader, ok := reader.Header["File"]
	if !ok {
		output.VerbosePrint("[!] Error: Client did not include filename in payload request.")
		http.Error(writer, "Filename required in payload request", http.StatusInternalServerError)
		return
	}
	filename := filenameReqHeader[0]
	platformHeader, ok := reader.Header["Platform"]
	if !ok {
		output.VerbosePrint("[!] Error: Client did not include platform in payload request.")
		http.Error(writer, "Platform required in payload request", http.StatusInternalServerError)
		return
	}
	platform := platformHeader[0]
	pawHeader, ok := reader.Header["Paw"]
	if !ok {
		output.VerbosePrint("[!] Error: Client did not include paw in payload request.")
		http.Error(writer, "Paw required in payload request", http.StatusInternalServerError)
		return
	}
	clientPaw := pawHeader[0]

	// Build profile to send request upstream.
	profile := make(map[string]interface{})
	profile["server"] = h.upstreamServer
	profile["platform"] = platform
	profile["paw"] = clientPaw
	payloadBytes, realFilename := h.upstreamComs.GetPayloadBytes(profile, filename)

	// Prepare response for client
	responseHeaders := make(map[string][]string)
	contentDispHeader := make([]string, 1)
	contentDispHeader[0] = fmt.Sprintf("attachment; filename=%s", realFilename)
	responseHeaders["CONTENT-DISPOSITION"] = contentDispHeader
	filenameRespHeader := make([]string, 1)
	filenameRespHeader[0] = realFilename
	responseHeaders["FILENAME"] = filenameRespHeader
	if err := sendResponseToClient(payloadBytes, responseHeaders, writer); err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error sending payload response to client: %s", err.Error()))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func sendResponseToClient(data []byte, headers map[string][]string, writer http.ResponseWriter) error {
	if headers != nil {
		for header, val := range headers {
			writer.Header().Set(header, val[0])
			for i := 1; i < len(val); i++ {
				writer.Header().Add(header, val[i])
			}
		}
	}
	_, err := writer.Write(data)
	return err
}

// Port must be set for the HTTP receiver before calling this method.
func (h *HttpReceiver) getReachableUrls() ([]string, error) {
	var urlList []string
	ipAddrs, err := h.getLocalIPv4Addresses()
	if err != nil {
		return nil, err
	}
	for _, addr := range ipAddrs {
		url := "http://" + addr + ":" + strconv.Itoa(h.port)
		urlList = append(urlList, url)
	}
	return urlList, nil
}

// Return list of local IPv4 addresses for this machine (exclude loopback and unspecified addresses)
func (h *HttpReceiver) getLocalIPv4Addresses() ([]string, error) {
	var localIpList []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addresses {
			var ipAddr net.IP
			switch v:= addr.(type) {
			case *net.IPNet:
				ipAddr = v.IP
			case *net.IPAddr:
				ipAddr = v.IP
			}
			if ipAddr != nil && !ipAddr.IsLoopback() && !ipAddr.IsUnspecified() && (ipAddr.To4() != nil) {
				localIpList = append(localIpList, ipAddr.String())
			}
		}
	}
	return localIpList, nil
}

// Helper method for initializing the receiver port.
func (h *HttpReceiver) initializeReceiverPort() error {
	// Try 5 random ports before giving up.
	retryCount := 0
	for (retryCount < maxPortRetries) {
		currPort := generateRandomPort(minReceiverPort, maxReceiverPort)
		currBindPortStr := ":" + strconv.Itoa(currPort)

		// Check if port is already in use.
		if ln, err := net.Listen("tcp", currBindPortStr); err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Error trying to use random port %d: %s", currPort, err.Error()))
		} else {
			h.port = currPort
			h.bindPortStr = currBindPortStr
			return ln.Close()
		}
	}
	return errors.New(fmt.Sprintf("Failed to find available port %d consecutive times.", maxPortRetries))
}

// Generate random port for the receiver in the range [minPort, maxPort]
func generateRandomPort(minPort int, maxPort int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(maxPort - minPort) + minPort
}