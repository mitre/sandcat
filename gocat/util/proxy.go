package util 

import (
	"fmt"
	"net"
	"net/http"
	"time"
	"io/ioutil"
	"bytes"
)

//StartProxy creates an HTTP listener to forward traffic to server
func StartProxy(server string) {
	 http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpClient := http.Client{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(body))
		url := fmt.Sprintf(server, r.RequestURI)

		proxyReq, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
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
	http.ListenAndServe(":8889", nil)
}

//FindProxy locates a useable host for coms
func FindProxy() string {
	//for _, ip := range calculateNetworkIPs(net.ParseIP(localIP)) {
    //    fmt.Println(ip)
	//}
	
	connected := testConnection("127.0.0.1")
	if connected {
		fmt.Println("[+] Located available proxy server", "127.0.0.1")
		return "http://127.0.0.1:8889"
	}
	return ""
}

func testConnection(ip string) bool {
	conn, _ := net.DialTimeout("tcp", net.JoinHostPort(ip, "8889"), time.Second)
	if conn != nil {
		defer conn.Close()
		return true
	}
	return false
}
