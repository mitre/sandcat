package proxy

import (
	"bytes"
	"context"
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

var httpProxyName = "HTTP"
var maxPortRetries = 5
var minReceiverPort = 50000
var maxReceiverPort = 63000

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
	// Make sure the agent uses HTTP with the C2.
	switch upstreamComs.(type) {
	case contact.API:
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
	default:
		return errors.New("Cannot initialize HTTP proxy receiver if agent is not using HTTP communication with the C2.")
	}
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
	switch newComs.(type) {
	case contact.API:
		h.upstreamComs = newComs
	default:
		output.VerbosePrint("[-] Cannot switch to non-HTTP comms.")
	}
}

func (h *HttpReceiver) GetReceiverAddresses() []string {
	return h.urlList
}

// Helper method for StartReceiver. Starts HTTP proxy to forward messages from peers to the C2 server.
func (h *HttpReceiver) startHttpProxy() {
	proxyHandler := func(writer http.ResponseWriter, reader *http.Request) {
		// Get data from the message that client peer sent.
		body, err := ioutil.ReadAll(reader.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		reader.Body = ioutil.NopCloser(bytes.NewReader(body))

		// Forward the request to the C2 server, and send back the response.
		resp, err := h.forwardRequestUpstream(body, writer, reader)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadGateway)
			output.VerbosePrint(fmt.Sprintf("[-] Error forwarding HTTP request: %s", err.Error()))
			return
		}
		if err = h.forwardResponseDownstream(resp, writer); err!= nil {
			http.Error(writer, err.Error(), http.StatusBadGateway)
			output.VerbosePrint(fmt.Sprintf("[-] Error forwarding HTTP response: %s", err.Error()))
		}
	}
	http.HandleFunc("/", proxyHandler)
	if err := http.ListenAndServe(h.bindPortStr, nil); err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] HTTP proxy error: %s", err.Error()))
	}
}

// Helper method for startHttpProxy that will forward the HTTP request upstream. Returns the response.
func (h *HttpReceiver) forwardRequestUpstream(body []byte, writer http.ResponseWriter, reader *http.Request) (*http.Response, error) {
	// Determine where to forward the request.
	url := h.upstreamServer + reader.RequestURI

	// Forward the request to the C2 server, and send back the response.
	httpClient := http.Client{}
	proxyReq, err := http.NewRequest(reader.Method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Copy headers received from client.
	proxyReq.Header = make(http.Header)
	for header, val := range reader.Header {
		proxyReq.Header[header] = val
	}
	return httpClient.Do(proxyReq)
}

func (h *HttpReceiver) forwardResponseDownstream(resp *http.Response, writer http.ResponseWriter) error {
	// Send back headers received from upstream.
	for header, val := range resp.Header {
		writer.Header().Set(header, val[0])
		for i := 1; i < len(val); i++ {
			writer.Header().Add(header, val[i])
		}
	}
	defer resp.Body.Close()
	bites, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_, err = writer.Write(bites)
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