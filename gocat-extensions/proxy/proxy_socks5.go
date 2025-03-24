package proxy

import (
	"fmt"
	"net"
	"sync"

	"github.com/armon/go-socks5"
	"github.com/mitre/gocat/contact"
	"github.com/mitre/gocat/output"
)

// SOCKS5Receiver implements the P2pReceiver interface.
type SOCKS5Receiver struct {
	listenAddr   string
	listener     net.Listener
	server       *socks5.Server
	upstreamComs *contact.Contact
	waitgroup    *sync.WaitGroup
	agentPaw     string // Store only the agent's PAW instead of a reference to Agent
	receiverName string
}

func init() {
	P2pReceiverChannels["socks5"] = &SOCKS5Receiver{}
}

// InitializeReceiver sets up the SOCKS5 server in memory and finds an open port.
func (s *SOCKS5Receiver) Initialize(ctx ReceiverInitContext) error {
	s.upstreamComs = ctx.UpstreamComs
	s.waitgroup = ctx.WaitGroup
	s.agentPaw = ctx.AgentPaw // Store PAW immediately
	s.receiverName = ctx.ReceiverName

	// Find an available port before running the receiver
	listener, err := net.Listen("tcp", "127.0.0.1:0") // OS assigns an available port
	if err != nil {
		return output.VerbosePrint(fmt.Sprintf("[-] SOCKS5 proxy failed to find an available port: %v", err))
	}
	s.listener = listener
	s.listenAddr = listener.Addr().String()

	// Create SOCKS5 server
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		listener.Close() // Ensure cleanup if the server fails to create
		return fmt.Errorf("[-] Failed to create in-memory SOCKS5 server: %v", err)
	}
	s.server = server
	return nil
}

// RunReceiver starts the SOCKS5 proxy listener.
func (s *SOCKS5Receiver) RunReceiver() {
	output.VerbosePrint(fmt.Sprintf("[DEBUG] SOCKS5 Proxy Receiver is attempting to start."))

	if s.listener == nil {
		output.VerbosePrint(fmt.Sprintf("[-] SOCKS5 proxy has no valid listener. Cannot start."))
		return
	}

	// Start the SOCKS5 server in-memory
	go func() {
		defer s.waitgroup.Done()
		if err := s.server.Serve(s.listener); err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] SOCKS5 proxy encountered an error: %v", err))
			s.Terminate() // Cleanup if failure occurs
		}
	}()
}

// Terminate stops the SOCKS5 server.
func (s *SOCKS5Receiver) Terminate() {
	output.VerbosePrint(fmt.Sprintf("[*] Shutting down in-memory SOCKS5 proxy..."))
	if s.server != nil {
		s.server = nil
	}
	if s.listener != nil {
		s.listener.Close()
		s.listenAddr = "" // Reset address
	}
}

// UpdateAgentPaw updates the PAW for the SOCKS5Receiver
func (s *SOCKS5Receiver) UpdateAgentPaw(newPaw string) {
	s.agentPaw = newPaw
}

// GetReceiverAddresses returns the listen address.
func (s *SOCKS5Receiver) GetReceiverAddresses() []string {
	if s.listenAddr == "" {
		return []string{} // Return an empty slice if no address assigned
	}
	return []string{s.listenAddr} // Returns the dynamically assigned port
}

func init() {
	P2pReceiverChannels["socks5"] = &SOCKS5Receiver{}
}
