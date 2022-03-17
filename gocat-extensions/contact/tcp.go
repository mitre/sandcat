package contact

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mitre/gocat/output"
)

type TCP struct {
	conn                net.Conn
	name                 string
	serverAddr           string
	serverIp             string
	serverPort           string
	instructionBucket    [][]byte
	payloadBucket        map[string]*payloadRecord
	outgoingBucket       [][]byte
	proxyToClientBucket  [][]byte
	proxyToServerBucket  [][]byte
	// payloadRequestBucket [][]byte
	// payloadBucket        map[string][]byte
}

type payloadRecord struct {
	sync.Mutex
	bytes     []byte
	waitCount int

	cond *sync.Cond
}

func newPayloadRecord() *payloadRecord {
	p := payloadRecord{}
	p.cond = sync.NewCond(&p)
	return &p
}

func init() {
	CommunicationChannels["tcp"] = TCP{}
}

func (t *TCP) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
	// For now, we'll just try to connect once, and quit if it fails
	addrParts := strings.Split(t.serverAddr, ":")
	if len(addrParts) != 2 {
		output.VerbosePrint("[!] Error - server address not correctly formatted. Must provide as IP:PORT")
		return false, nil
	}

	t.serverIp = addrParts[0]
	t.serverPort = addrParts[1]
	conn, err := net.Dial("tcp", t.serverPort)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] %s", err))
	}
	t.conn = conn

	handshake(conn, profile)
	output.VerbosePrint(fmt.Sprintf("[+] TCP established for %s", profile["paw"]))

	go t.listenAndHandleIncoming(profile)
	go handleOutgoing()
	go handleProxy()
	return true, nil

}

func handshake(conn net.Conn, profile map[string]interface{}) {
	/*
		Sends the initial beacon to the server after creating the connection. Retrieves a paw.
	*/
	//write the profile
	jdata, _ := json.Marshal(profile)
	conn.Write(jdata)
	conn.Write([]byte("\n"))

	//read back the paw
	data := make([]byte, 512)
	n, _ := conn.Read(data)
	paw := string(data[:n])
	// conn.Write([]byte("\n"))
	profile["paw"] = strings.TrimSpace(string(paw))
}

func (t *TCP) listenAndHandleIncoming(profile map[string]interface{}) {
	/*
		This function continually listens on the TCP connection for information from the server.
		When a message is received from the server, the type of message is checked, and then the contents are
		appended to the appropriate bucket.
		Currently, there are 2 types of messages: Instructions and Proxy.
	*/

	scanner := bufio.NewScanner(t.conn)
	for {
		for scanner.Scan() {
			var messageWrapper map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &messageWrapper); err != nil {
				output.VerbosePrint(fmt.Sprintf("[-] Malformed TCP message received: %s", err.Error()))
			} else {
				if messageWrapper["messageType"] == "instruction" {
					t.instructionBucket = append(t.instructionBucket, []byte(messageWrapper["message"].(string)))
				} else if messageWrapper["messageType"] == "proxy" {
					t.proxyToClientBucket = append(t.proxyToClientBucket, []byte(messageWrapper["message"].(string)))
				} else if messageWrapper["messageType"] == "payload"{
					handlePayloadResponse(messageWrapper["message"])
				} else {
					output.VerbosePrint(fmt.Sprintf("[-] TCP Message Type not recognized: %s", messageWrapper["messageType"]))
				}
			}

			// bites, status, commandTimestamp := commands.RunCommand(strings.TrimSpace(message), server, profile)
			// pwd, _ := os.Getwd()
			// response := make(map[string]interface{})
			// response["response"] = string(bites)
			// response["status"] = status
			// response["pwd"] = pwd
			// response["agent_reported_time"] = util.GetFormattedTimestamp(commandTimestamp, "2006-01-02T15:04:05Z")
			// jdata, _ := json.Marshal(response)
			// conn.Write(jdata)
		}
	}
}

func handlePayloadResponse(payloadMessage []byte) {
	/*
		This function processes received Payload responses. It stores the payload bytes returned into the appropriate
			payloadRec, and then broadcasts to all waiting payload requests that the payload has been retrieved.
	*/
	var payloadResponse map[string]interface{}
	if err := json.Unmarshal(payloadMessage, &payloadResponse) {
		output.VerbosePrint(fmt.Sprintf("[-] Malformed TCP message received: %s", err.Error()))
	} else {
		if payloadName, ok := payloadResponse["filename"]; !ok {
			output.VerbosePrint(fmt.Sprintf("[-] Payload response did not include filename"))
		}
		if payloadRec, ok := payloadBucket[payloadName]; !ok {
			output.VerbosePrint(fmt.Sprintf("[-] Payload returned, but no payload record for: %s", payloadName))
		}
		if payloadBytes, ok := payloadResponse["bytes"]; !ok {
			output.VerbosePrint(fmt.Sprintf("[-] Payload response did not include file bytes for: %s", payloadName))
		}

		payloadRec.Lock()
		payloadRec.bytes = payloadBytes
		payloadRec.cond.Broadcast()
		payloadRec.Unlock()
	}
}

func (t *TCP) handleOutgoing() {
	/*
		This function keeps going through payloadBucket and outgoingBucket, and sends the entries in each to the server.
	*/
	for true {
		for len(t.outgoingBucket) > 0 {
			response := t.outgoingBucket[0]
			t.conn.Write(response)
			t.outgoingBucket = t.outgoingBucket[1:]
		}
	}
}

func (t *TCP) handleProxy() {
	
}

func (t *TCP) GetBeaconBytes(profile map[string]interface{}) []byte {
	/*
		Checks for beacon in instructionBucket.
		If there is at least one beacon, removes and returns first one
	*/
	if len(t.instructionBucket) > 0 {
		beaconBytes := t.instructionBucket[0]
		t.instructionBucket = t.instructionBucket[1:]
		return beaconBytes
	}

	return nil
}

func (t *TCP) GetPayloadBytes(profile map[string]interface{}, payload string) ([]byte, string) {
	/*
		Gets payload name and places in payloadBucket.
		Once handleOutgoing processes payload and retrieves payload bytes, this function returns those bytes.
		We keep track of how many instances a payload is requested, so the payload is deleted from the bucket only
			when there are no more requests for that payload.
	*/
	var payloadRec *payloadRecord
	if payloadRec, ok := t.payloadBucket[payload]; ok {
		//payload request already exists, so we'll add another consumer
		payloadRec.waitCount += 1
	} else {
		//payload request does not exist yet, we need to make a new one, and populate request fields
		payloadRec = newPayloadRecord()
		payloadRec.waitCount = 1
		t.payloadBucket[payload] = payloadRec

		requestFields := make(map[string]interface{})
		requestFields["messageType"] = "payloadRequest"
		requestFields["payload"] = payload
		requestFields["paw"] = profile["paw"]
		requestFields["platform"] = profile["platform"]

		request, err := json.Marshal(requestFields)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Cannot send payload request. Error with profile marshal: %s", err.Error()))
		} else {
			append(t.outgoingBucket, request)
		}
	}

	payloadRec.Lock()
	payloadRec.cond.Wait()
	payloadRec.Unlock()

	payloadBytes := payloadRec.bytes

	if payloadRec.waitCount == 1 {
		// we are the only request waiting for the payload, so now that we have the payload we can delete the entry
		delete(t.payloadBucket, payload)
	} else {
		payloadRec.Lock()
		// there are other requests waiting for the payload, so we'll just decrement the counter
		payloadRec.waitCount -= 1
		payloadRec.Unlock()
	}

	return payloadBytes, payloadName
}

func (t *TCP) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	/*
		This function adds execution output to the outgoingBucket for handleOutgoing() to process
	*/

	profileCopy := make(map[string]interface{})
	for k, v := range profile {
		profileCopy[k] = v
	}
	profileCopy["messageType"] = "executionResults"
	results := make([]map[string]interface{}, 1)
	results[0] = result
	profileCopy["results"] = results

	data, err := json.Marshal(profileCopy)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send results. Error with profile marshal: %s", err.Error()))
	} else {
		append(t.outgoingBucket, data)
	}
}

func (t *TCP) GetName() string {
	return t.name
}

func (t *TCP) SetUpstreamDestAddr(upstreamDestAddr string) {
	t.serverAddr = upstreamDestAddr
}

func (t *TCP) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	/*
		This function generates a file upload request and appends it to outgoingBucket.
		handleOutgoing then processes the request.
	*/

	upload := make(map[string]interface{})
	upload["messageType"] = "fileUpload"
	upload["filename"] = uploadName
	upload["data"] = data

	request, err := json.Marshal(upload)
	if err != nil {
		return err)
	} else {
		append(t.outgoingBucket, data)
	}
	return nil
}

func (t *TCP) SupportsContinuous() bool {
	return true
}

func getRandomId() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}
