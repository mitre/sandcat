package proxy

import (
    "encoding/json"
    "../contact"
)

// Define MessageType values for P2pMessage
const (
	GET_INSTRUCTIONS = 1
	GET_PAYLOAD_BYTES = 2
	SEND_EXECUTION_RESULTS = 3
	RESPONSE_INSTRUCTIONS = 4
	RESPONSE_PAYLOAD_BYTES = 5
	RESPONSE_SEND_EXECUTION_RESULTS = 6
	RESEND_REQUEST = 7 // For server to ask the client to resent the request to a new specified destination.
)

//P2pReceiver defines required functions for relaying messages between peers and an upstream peer/c2.
type P2pReceiver interface {
	StartReceiver(profile map[string]interface{}, p2pReceiverConfig map[string]string, upstreamComs contact.Contact)
}

// Defines message structure for p2p
type P2pMessage struct {
    RequestingAgentPaw string // Paw of agent sending the original request.
    SendResponseTo string // return address for responses
    MessageType int
    Payload []byte
    Populated bool
}

// P2pReceiverChannels contains the P2pReceiver implementations
var P2pReceiverChannels = map[string]P2pReceiver{}

// Helper Functions

// Build p2p message and return the bytes of its JSON marshal.
func BuildP2pMsgBytes(paw string, messageType int, payload []byte, respondTo string) []byte {
    p2pMsg := &P2pMessage{
        RequestingAgentPaw: paw,
        SendResponseTo: respondTo,
        MessageType: messageType,
        Payload: payload,
        Populated: true,
    }
    p2pMsgData, _ := json.Marshal(p2pMsg)
    return p2pMsgData
}

// Convert bytes of JSON marshal into P2pMessage struct
func BytesToP2pMsg(data []byte) P2pMessage {
    var message P2pMessage
    if err := json.Unmarshal(data, &message); err == nil {
        return message
    }
    return P2pMessage{}
}

// Check if message is empty.
func MsgIsEmpty(msg P2pMessage) bool {
    return !msg.Populated
}