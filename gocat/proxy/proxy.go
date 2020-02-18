package proxy

import (
    "encoding/json"
    "../contact"
)

// Define MessageType for P2pMessage
const (
	INSTR_GET_INSTRUCTIONS = 0
	INSTR_GET_PAYLOAD_BYTES = 1
	INSTR_SEND_EXECUTION_RESULTS = 2
	RESEND_REQUEST = 3 // For server to ask the client to resent the request to a new specified destination.
	RESPONSE_INSTRUCTIONS = 4 // For server to send instructions to client
	RESPONSE_PAYLOAD_BYTES = 5
	RESPONSE_SEND_EXECUTION_RESULTS = 6
)

//P2pReceiver defines required functions for relaying messages between peers and an upstream peer/c2.
type P2pReceiver interface {
	StartReceiver(profile map[string]interface{}, p2pReceiverConfig map[string]string, upstreamComs contact.Contact)
}


// Defines message structure for p2p
type P2pMessage struct {
    RequestingAgentPaw string
    MessageType int
    Payload []byte
    Populated bool
}

// Helper Functions

// Build p2p message and return the bytes of its JSON marshal.
func buildP2pMsgBytes(paw string, messageType int, payload []byte) []byte {
    p2pMsg := make(map[string]interface{})
    p2pMsg["RequestingAgentPaw"] = paw
    p2pMsg["MessageType"] = messageType
    p2pMsg["Payload"] = payload
    p2pMsg["Populated"] = true
    p2pMsgData, _ := json.Marshal(p2pMsg)

    return p2pMsgData
}

// Convert bytes of JSON marshal into P2pMessage struct
func bytesToP2pMsg(data []byte) P2pMessage {
    var message P2pMessage
    json.Unmarshal(data, &message)

    return message
}

// Check if message is empty.
func msgIsEmpty(msg P2pMessage) bool {
    return !msg.Populated
}

// P2pReceiverChannels contains the P2pReceiver implementations
var P2pReceiverChannels = map[string]P2pReceiver{}