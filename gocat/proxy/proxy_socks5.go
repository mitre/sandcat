package proxy

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/armon/go-socks5"
	"github.com/mitre/gocat/contact"
)

// SOCKS5Receiver implements the P2pReceiver interface.
type SOCKS5Receiver struct {
	listenAddr   string
	server       *socks5.Server
	upstreamComs *contact.Contact
	waitgroup    *sync.WaitGroup
	agentPaw     string
	active       bool // ✅ Tracks if the proxy is currently active
}

// InitializeReceiver sets up the SOCKS5 receiver but does NOT start it.
func (s *SOCKS5Receiver) InitializeReceiver(agentServer *string, upstreamComs *contact.Contact, waitgroup *sync.WaitGroup, agentPaw string) error {
	s.upstreamComs = upstreamComs
	s.waitgroup = waitgroup
	s.agentPaw = agentPaw
	s.active = false // Proxy is inactive at initialization

	// Create SOCKS5 server
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		return fmt.Errorf("[-] Failed to create in-memory SOCKS5 server: %v", err)
	}
	s.server = server

	// ✅ Start listening for proxy activation requests
	go s.listenForProxyMessage()

	return nil
}

// listenForProxyMessage waits for an activation request before starting the proxy.
func (s *SOCKS5Receiver) listenForProxyMessage() {
	for {
		time.Sleep(2 * time.Second) // Poll every 2 seconds

		if contact.ShouldActivateProxy() { // ✅ External function to check activation
			s.RunReceiver()
		} else if s.active && contact.ShouldDeactivateProxy() {
			s.Terminate()
		}
	}
}

// RunReceiver starts the SOCKS5 proxy dynamically.
func (s *SOCKS5Receiver) RunReceiver() {
	if s.active {
		log.Println("[!] SOCKS5 proxy is already running at", s.listenAddr)
		return
	}

	log.Println("[DEBUG] SOCKS5 Proxy Receiver is attempting to start.")

	// Find an open port dynamically
	listener, err := net.Listen("tcp", "127.0.0.1:0") // OS assigns an available port
	if err != nil {
		log.Printf("[-] SOCKS5 proxy failed to find an available port: %v", err)
		return
	}

	// ✅ Set listen address before starting
	s.listenAddr = listener.Addr().String()
	s.active = true

	log.Printf("[+] SOCKS5 proxy dynamically assigned to %s", s.listenAddr)

	// ✅ Notify Sandcat that the proxy is running
	s.sendMessageToSandcat("proxy_active", s.listenAddr)

	// Start the SOCKS5 server
	s.waitgroup.Add(1)
	go func() {
		defer s.waitgroup.Done()
		if err = s.server.Serve(listener); err != nil {
			log.Printf("[-] SOCKS5 proxy encountered an error: %v", err)
			s.active = false
			s.listenAddr = ""
		}
	}()
}

// Terminate stops the SOCKS5 proxy.
func (s *SOCKS5Receiver) Terminate() {
	if !s.active {
		log.Println("[!] SOCKS5 proxy is not running.")
		return
	}

	log.Println("[*] Shutting down in-memory SOCKS5 proxy...")
	if s.server != nil {
		fmt.Println("[*] SOCKS5 proxy stopped.")
	}
	s.active = false
	s.listenAddr = ""

	// ✅ Notify Sandcat that the proxy is inactive
	s.sendMessageToSandcat("proxy_inactive", "")
}

// sendMessageToSandcat notifies Sandcat of the proxy status.
func (s *SOCKS5Receiver) sendMessageToSandcat(status string, address string) {
	proxyStatus := map[string]string{
		"status":  status,
		"address": address,
	}
	contact.SendProxyStatus(proxyStatus) // ✅ Function to send data to Sandcat
}

// UpdateAgentPaw updates the PAW for the SOCKS5Receiver
func (s *SOCKS5Receiver) UpdateAgentPaw(newPaw string) {
    s.agentPaw = newPaw
}

// GetReceiverAddresses returns the listen address.
func (s *SOCKS5Receiver) GetReceiverAddresses() []string {
    if s.listenAddr == "" {
        return []string{"Inactive - Waiting for activation"} // ✅ Avoids empty output
    }
    return []string{s.listenAddr}
}

// ✅ Add IsRunning() method to check if the proxy is active
func (s *SOCKS5Receiver) IsRunning() bool {
	return s.active
}

// Initialize SOCKS5 proxy in the proxy channel list.
func init() {
	P2pReceiverChannels["socks5"] = &SOCKS5Receiver{}
}
