// +build windows

/*
 * This file contains implementations for a P2P client (SmbPipeAPI) and P2P receiver (SmbPipeReceiver)
 * Agents using SMB pipes to communicate upstream will do so using the P2P client, and the upstream agents receiving
 * the SMB pipe messages will need to have the SMB P2P receivers running. An agent can use both the SmbPipeAPI p2p
 * client and the SmbPipeReceiver at the same time, in which case it would act as an SMB forwarder, using SMB for
 * both upstream and downstream communications.
 *
 * Each agent using SMB Pipe P2P will have one SmbPipeAPI struct that implements the contact.Contact interface to
 * provide full communication between the agent and the upstream C2 (get payloads, get instructions, etc).
 * The SMB pipe API client assumes that it is talking to an SMB Pipe p2p receiver that knows how to process the
 * client messages accordingly. When using the SMB Pipe P2P client, the agent will generate a random pipe path
 * to listen on - this pipe will receive response messages when the agent sends requests upstream for itself.
 * But if an agent has downstream agents to forward messages for, those requests will also flow through the
 * SMB pipe API client. Thus, the agent will open up new random pipe paths for each downstream agent that it is
 * servicing, and the SmbPipeAPI struct will keep track of which pipe path is for which downstream agent so that
 * it can forward the responses appropriately.
 *
 * Each agent listening for SMB pipe P2P messages will activate an SMB pipe receiver that listens on a particular
 * pipe path. This pipe path is randomly generated using the hostname as the seed, so that a client agent
 * can calculate the pipe path for its upstream agent.  As the SMB pipe receiver gets messages from downstream
 * agents, it will forward the message upstream and return the response.  Each upstream P2P message via SMB pipes
 * must contain the requesting agent's pipe path for response messages, so the P2P receiver knows where to send
 * the responses.
 */

package proxy

import (
	"context"
	"fmt"
	"net"
	"encoding/json"
	"time"
	"regexp"
	"os"
	"sync"
	"github.com/mitre/gocat/output"
	"github.com/mitre/gocat/contact"

	_ "github.com/mitre/gocat/execute/donut" // necessary to initialize all submodules
	_ "github.com/mitre/gocat/execute/shells" // necessary to initialize all submodules
	_ "github.com/mitre/gocat/execute/shellcode" // necessary to initialize all submodules

	//"gopkg.in/natefinch/npipe.v2"
)

var (
	// lock for SMBPipeAPI client when editing the ReturnMailBoxPipePaths and ReturnMailBoxListeners for
	// API users (this agent and any downstream agents reaching out to this agent via P2P)
	// Needed because multiple go routines will use the same SmbPipeAPI if the agent is acting
	// as a receiver for multiple client agents, and the upstream comms for the receiver is type SmbPipeAPI.
	apiClientMutex sync.Mutex

	// For writes to the upstream pipe.
	upstreamPipeLock sync.Mutex
)

// Pipe-related constants.
const (
	pipeCharacters = "abcdefghijklmnopqrstuvwxyz1234567890"
	numPipeCharacters = int64(len(pipeCharacters))
	clientPipeNameMinLen = 10
	clientPipeNameMaxLen = 15
	maxChunkSize = 5*4096 // chunk size for writing to pipes.
	pipeDialTimeoutSec = 10 // number of seconds to wait before timing out of pipe dial attempt.
)

// SmbPipeAPI communicates through SMB named pipes. Implements the Contact interface.
type SmbPipeAPI struct {
	// Maps agent paws to full pipe paths for receiving forwarded responses on their behalf.
	returnMailBoxPipePaths map[string]string

	// Maps agent paws to Listener objects for the corresponding local pipe paths.
	returnMailBoxListeners map[string]net.Listener
	name string
}

//PipeReceiver forwards data received from SMB pipes to the upstream server. Implements the P2pReceiver interface
type SmbPipeReceiver struct {
	mainPipeName string
	localMainPipePath string // full pipe path from a local perrspective. \\.\pipe\<pipename>
	externalMainPipePath string // full pipe path from an external perspective. \\hostname\pipe\<pipename>
	upstreamComs contact.Contact // Contact implementation to handle upstream communication.
	listener net.Listener // Listener object for this receiver.
	upstreamServer string // Location of upstream server to send data to.
	waitgroup *sync.WaitGroup
	receiverContext context.Context
	receiverCancelFunc context.CancelFunc
}

// Auxiliary struct that defines P2P message payload structure for an ability payload request
type payloadRequestInfo struct {
	PayloadName string
	Profile map[string]interface{}
}

// Auxiliary struct that defines P2P message payload structure for an ability payload response
type payloadResponseInfo struct {
	PayloadName string
	PayloadData []byte
}

func init() {
	//P2pClientChannels["SmbPipe"] = &SmbPipeAPI{
	contact.CommunicationChannels["SmbPipe"] = &SmbPipeAPI{
		make(map[string]string),
		make(map[string]net.Listener),
		"SmbPipe",
	}
	P2pReceiverChannels["SmbPipe"] = &SmbPipeReceiver{}
}

// SmbPipeReceiver Implementation (implements P2pReceiver interface).

func (s *SmbPipeReceiver) InitializeReceiver(server string, upstreamComs contact.Contact, waitgroup *sync.WaitGroup) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	s.mainPipeName = getMainPipeName(hostname)
	s.localMainPipePath = "\\\\.\\pipe\\" + s.mainPipeName
	s.externalMainPipePath = "\\\\" + hostname + "\\pipe\\" + s.mainPipeName
	s.listener, err = listenPipeFullAccess(s.localMainPipePath)
	if err != nil {
		return err
	}
	s.upstreamServer = server
	s.upstreamComs = upstreamComs
	s.waitgroup = waitgroup
	s.receiverContext, s.receiverCancelFunc = context.WithTimeout(context.Background(), 5*time.Second)
	return nil
}


// Listen on agent's main pipe for client connection. This method must be run as a go routine.
func (s *SmbPipeReceiver) RunReceiver() {
	output.VerbosePrint(fmt.Sprintf("[*] Starting SMB pipe proxy receiver on local pipe path %s", s.localMainPipePath))
	output.VerbosePrint(fmt.Sprintf("[*] SMB pipe proxy receiver has upstream contact at %s", s.upstreamServer))
	s.startReceiverHelper()
}

func (s *SmbPipeReceiver) UpdateUpstreamServer(newServer string) {
	s.upstreamServer = newServer
}

func (s *SmbPipeReceiver) UpdateUpstreamComs(newComs contact.Contact) {
	s.upstreamComs = newComs
}

func (s *SmbPipeReceiver) Terminate() {
	defer func() {
		s.waitgroup.Done()
		s.receiverCancelFunc()
	}()
	if err := s.listener.Close(); err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error when closing pipe listener: %s", err.Error()))
	}
}

func (s *SmbPipeReceiver) GetReceiverAddresses() []string {
	addrList := make([]string, 1)
	addrList[0] = s.externalMainPipePath
	return addrList
}

// Helper method for StartReceiver.
func (s *SmbPipeReceiver) startReceiverHelper() {
	// Whenever a client connects to pipe with a request, process the request using a go routine.
	for {
		totalData, err := fetchDataFromPipe(s.listener)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error with reading client input: %s", err.Error()))
			continue
		}
		message, err := bytesToP2pMsg(totalData)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error converting data to p2p message: %s", err.Error()))
		} else if msgIsEmpty(message) {
			output.VerbosePrint("[!] Error - downstream agent sent an empty P2P message.")
			continue
		}
		switch message.MessageType {
			case GET_INSTRUCTIONS:
				go s.forwardGetBeaconBytes(message)
			case GET_PAYLOAD_BYTES:
				go s.forwardPayloadBytesDownload(message)
			case SEND_EXECUTION_RESULTS:
				go s.forwardSendExecResults(message)
			default:
				output.VerbosePrint(fmt.Sprintf("[!] ERROR: invalid instruction type for receiver-bound p2p message: %d", message.MessageType))
		}
	}
}

// Pass the beacon request to the upstream server, and return the response.
func (s *SmbPipeReceiver) forwardGetBeaconBytes(message P2pMessage) {
    // Message payload contains profile to send upstream
    clientProfile := make(map[string]interface{})
    if err := json.Unmarshal(message.Payload, &clientProfile); err != nil {
    	output.VerbosePrint(fmt.Sprintf("[!] Error extracting client profile from p2p message: %s", err.Error()))
    	return
    }
    clientProfile["server"] = s.upstreamServer // make sure we send the instructions to the right place.
    output.VerbosePrint(fmt.Sprintf("[*] Forwarding instructions request to on behalf of paw %s", message.SourcePaw))
    response := s.upstreamComs.GetBeaconBytes(clientProfile)

    // Connect to client mailbox to send response back to client.
    if len(message.SourceAddress) > 0 {
        pipeMsgData, err := buildP2pMsgBytes("", RESPONSE_INSTRUCTIONS, response, "")
        if err != nil {
        	output.VerbosePrint(fmt.Sprintf("[!] Error building response message for client: %s", err.Error()))
    		return
        }
        if _, err = sendDataToPipe(message.SourceAddress, pipeMsgData); err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error sending response message to client: %s", err.Error()))
    		return
        }
        output.VerbosePrint(fmt.Sprintf("[*] Sent beacon response to paw %s", message.SourcePaw))
    } else {
        output.VerbosePrint(fmt.Sprintf("[-] ERROR. P2p message from client did not specify a return address."))
    }
}

// Pass the payload bytes download request to the upstream server, and return the response.
func (s *SmbPipeReceiver) forwardPayloadBytesDownload(message P2pMessage) {
    // Message payload contains client profile and requested payload name.
    var requestInfo payloadRequestInfo
    if err := json.Unmarshal(message.Payload, &requestInfo); err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error extracting payload request info from p2p message: %s", err.Error()))
    	return
    }

    // Get upstream response, which contains payload data and name, and forward response to client.
    if requestInfo.Profile == nil {
    	output.VerbosePrint("[!] Error - client did not send profile information in payload request.")
    	return
    }
    if len(requestInfo.PayloadName) == 0 {
    	output.VerbosePrint("[!] Error - client did not send payload name in payload request.")
    	return
    }
   	requestInfo.Profile["server"] = s.upstreamServer // Make sure request gets sent to the right place.
    output.VerbosePrint(fmt.Sprintf("[*] Forwarding payload bytes request for payload %s on behalf of paw %s", requestInfo.PayloadName, message.SourcePaw))
    payloadData, payloadName := s.upstreamComs.GetPayloadBytes(requestInfo.Profile, requestInfo.PayloadName)
    respInfo := payloadResponseInfo{
    	PayloadData: payloadData,
    	PayloadName: payloadName,
    }
    respData, err := json.Marshal(respInfo)
    if err != nil {
    	output.VerbosePrint(fmt.Sprintf("[!] Error marshaling payload response info: %s", err.Error()))
    	return
    }
    if len(message.SourceAddress) > 0 {
        pipeMsgData, err := buildP2pMsgBytes("", RESPONSE_PAYLOAD_BYTES, respData, "")
        if err != nil {
        	output.VerbosePrint(fmt.Sprintf("[!] Error building payload response message: %s", err.Error()))
    		return
        }
        sendDataToPipe(message.SourceAddress, pipeMsgData)
        output.VerbosePrint(fmt.Sprintf("[*] Sent %d payload bytes for payload %s to client paw %s", len(payloadData), payloadName, message.SourcePaw))
    } else {
        output.VerbosePrint(fmt.Sprintf("[-] ERROR. P2p message from client did not specify a return address."))
    }
}

func (s *SmbPipeReceiver) forwardSendExecResults(message P2pMessage) {
    // message payload contains client profile and result info.
    clientProfile := make(map[string]interface{})
    if err := json.Unmarshal(message.Payload, &clientProfile); err != nil {
    	output.VerbosePrint(fmt.Sprintf("[!] Error extracting execution result info from p2p message: %s", err.Error()))
    	return
    }
    resultInfo, ok := clientProfile["results"]
    if !ok {
    	output.VerbosePrint("[!] Error. Client did not include execution results.")
        return
    }
    result := resultInfo.([]interface{})[0]
    clientProfile["server"] = s.upstreamServer

    // Send execution results upstream. No response will be sent to client.
    output.VerbosePrint(fmt.Sprintf("[*] Forwarding execution results on behalf of paw %s", message.SourcePaw))
    s.upstreamComs.SendExecutionResults(clientProfile, result.(map[string]interface{}))
}

/*
 * SmbPipeAPI implementation
 */

// Contact API functions

func (s *SmbPipeAPI) GetBeaconBytes(profile map[string]interface{}) []byte {
	requestingPaw := getPawFromProfile(profile)
	mailBoxPipePath, mailBoxListener, err := s.fetchClientMailBoxInfo(requestingPaw, true)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error obtaining mailbox and listener for client paw %s: %s", requestingPaw, err.Error()))
		return nil
	}
	output.VerbosePrint(fmt.Sprintf("[*] P2P Client: going to fetch beacon bytes for paw %s", requestingPaw))

	// Send instruction request
	payload, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error marshaling profile: %s", err.Error()))
		return nil
	}
	upstreamPipeLock.Lock()
	err = sendRequestToServer(profile["server"].(string), requestingPaw, GET_INSTRUCTIONS, payload, mailBoxPipePath)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error sending beacon request to server: %s", err.Error()))
		upstreamPipeLock.Unlock()
		return nil
	}

	// Process response.
	respMessage, err := getResponseMessage(mailBoxListener)
	upstreamPipeLock.Unlock()
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error obtaining instruction response from server: %s", err.Error()))
		return nil
	} else if msgIsEmpty(respMessage) {
		output.VerbosePrint("[!] Error. Empty message from server.")
		return nil
	} else if respMessage.MessageType != RESPONSE_INSTRUCTIONS {
		output.VerbosePrint(fmt.Sprintf("[!] Error: server sent invalid response type for getting instructions: %d", respMessage.MessageType))
		return nil
	}
	// Message payload contains beacon bytes.
	return respMessage.Payload
}

// Will obtain the payload bytes in memory to be written to disk later by caller.
func (s *SmbPipeAPI) GetPayloadBytes(profile map[string]interface{}, payload string) ([]byte, string) {
    server := profile["server"].(string)
    paw := profile["paw"].(string)

    // Set up mailbox pipe and listener if needed.
	mailBoxPipePath, mailBoxListener, err := s.fetchClientMailBoxInfo(paw, true)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error obtaining mailbox and listener for client paw %s: %s", paw, err.Error()))
		return nil, ""
	}

	// Download payload bytes for a single payload. Create SMB Pipe message with
	// payload as a map[string]interface{} specifying the file name and agent profile.
	requestInfo := payloadRequestInfo{
		PayloadName: payload,
		Profile: profile,
	}
	msgPayload, err := json.Marshal(requestInfo)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error marshalling payload request info: %s", err.Error()))
		return nil, ""
	}
	output.VerbosePrint(fmt.Sprintf("[*] P2p Client Downloading new payload: %s", payload))
	upstreamPipeLock.Lock()
	err = sendRequestToServer(server, paw, GET_PAYLOAD_BYTES, msgPayload, mailBoxPipePath)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error sending payload request to server: %s", err.Error()))
		upstreamPipeLock.Unlock()
		return nil, ""
	}

	// Process response.
	respMessage, err := getResponseMessage(mailBoxListener)
	upstreamPipeLock.Unlock()
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error obtaining payload response from server: %s", err.Error()))
		return nil, ""
	} else if msgIsEmpty(respMessage) {
		output.VerbosePrint("[!] Error: server sent back empty message for payload request.")
	} else if respMessage.MessageType != RESPONSE_PAYLOAD_BYTES {
		output.VerbosePrint(fmt.Sprintf("[!] Error: server sent invalid response type for getting getting payload bytes: %d", respMessage.MessageType))
		return nil, ""
	}

	// Message payload contains payload bytes and true filename.
	var responseInfo payloadResponseInfo
	if err = json.Unmarshal(respMessage.Payload, &responseInfo); err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error unmarshalling payload response info: %s", err.Error()))
    	return nil, ""
	}
    if len(responseInfo.PayloadName) == 0 {
    	output.VerbosePrint("[!] Error. Server did not send payload name.")
        return nil, ""
    }
	return responseInfo.PayloadData, responseInfo.PayloadName
}

// Check if current server is a full pipe path. If not, return config with new pipe path using
// current server value and default generated pipe name.
func (s *SmbPipeAPI) C2RequirementsMet(profile map[string]interface{}, criteria map[string]string) (bool, map[string]string) {
	currentServer := profile["server"].(string)
	match, err := regexp.MatchString(`^\\\\[^\\]+\\pipe\\[^\\]+$`, currentServer)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Regular expression error: %s", err.Error()))
		return false, nil
	} else if !match {
		config := make(map[string]string)
		config["server"] = "\\\\" + currentServer + "\\pipe\\" + getMainPipeName(currentServer)
		return true, config
	}
    return true, nil
}

func (s *SmbPipeAPI) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	requestingPaw := getPawFromProfile(profile)

	// Set up mailbox pipe and listener if needed.
	mailBoxPipePath, _, err := s.fetchClientMailBoxInfo(requestingPaw, true)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] ERROR setting up mailbox listener: %s", err.Error()))
	}

	// Build SMB pipe message for sending execution results.
	// payload will contain JSON marshal of profile, with execution results
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := make([]map[string]interface{}, 1)
	results[0] = result
	profileCopy["results"] = results
	msgPayload, err := json.Marshal(profileCopy)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send results. Error with profile marshal: %s", err.Error()))
	}
	output.VerbosePrint(fmt.Sprintf("[*] P2p Client: sending execution results to %s", profile["server"].(string)))
	upstreamPipeLock.Lock()
	err = sendRequestToServer(profile["server"].(string), requestingPaw, SEND_EXECUTION_RESULTS, msgPayload, mailBoxPipePath)
	upstreamPipeLock.Unlock()
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error sending execution results to server: %s", err.Error()))
	}
}

func (s *SmbPipeAPI) GetName() string {
	return s.name
}