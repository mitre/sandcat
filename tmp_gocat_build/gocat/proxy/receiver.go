package proxy

import (
	"sync"
	"github.com/mitre/gocat/contact"
)

// InitContext is passed to all proxy receiver implementations during initialization
type ReceiverInitContext struct {
	AgentServer  *string
	UpstreamComs *contact.Contact
	WaitGroup    *sync.WaitGroup
	AgentPaw     string
	ReceiverName string
}

