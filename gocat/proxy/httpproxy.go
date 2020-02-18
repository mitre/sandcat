package proxy

import (
    "fmt"
    "net/http"
    "io/ioutil"
    "bytes"
    "../output"
    "../contact"
)

//HttpReceiver forwards data received from HTTP requests to the upstream server via HTTP. Implements the P2pReceiver interface.
type HttpReceiver struct { }

func init() {
	P2pReceiverChannels["http"] = HttpReceiver{}
}

// HttpReceiver Implementation. Assumes Agent can reach C2 server via HTTP

// Listen on port for client connection.
func (receiver HttpReceiver) StartReceiver(profile map[string]interface{}, p2pReceiverConfig map[string]string, upstreamComs contact.Contact) {
    switch upstreamComs.(type) {
    case contact.API:
        go receiver.startReceiverHelper(profile, p2pReceiverConfig["p2pReceiver"])
    default:
        output.VerbosePrint(fmt.Sprintf("[-] Cannot start HTTP proxy receiver if agent is not using HTTP communication with the C2."))
    }
}

// Helper method for StartReceiver. Must be run as a go routine.
func (receiver HttpReceiver) startReceiverHelper(profile map[string]interface{}, portStr string) {
    listenPort := ":" + portStr
    server := profile["server"].(string)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpClient := http.Client{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(body))
		url := server + r.RequestURI

		proxyReq, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			output.VerbosePrint(err.Error())
			return
		}
		proxyReq.Header = make(http.Header)
		for h, val := range r.Header {
			proxyReq.Header[h] = val
		}
		resp, err := httpClient.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		bites, _ := ioutil.ReadAll(resp.Body)
		w.Write(bites)
	})
	http.ListenAndServe(listenPort, nil)
}