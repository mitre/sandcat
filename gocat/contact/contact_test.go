package contact_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mitre/gocat/contact"
)

var continuousContacts = []string{"tcp"}

func TestSupportsContinuous(t *testing.T) {
	for contactName, contactImpl := range contact.CommunicationChannels {
		t.Log(contactName)
		var want bool
		if contains(continuousContacts, contactName) {
			want = true
		} else {
			want = false
		}
		result := contactImpl.SupportsContinuous()
		if want != result {
			t.Errorf("%s SupportsContinuous() should return %s, but returned %s", contactName, strconv.FormatBool(want), strconv.FormatBool(result))
		}
	}

}

func TestTCPC2RequirementsMet(t *testing.T) {
	if tcpContact, ok := contact.CommunicationChannels["tcp"]; ok {
		listener := setupTCPListener(t, tcpContact)
		defer listener.Close()

		profile, c2Config := setupProfileAndConfig()

		connChan := make(chan net.Conn, 1)
		go handleTCPHandshake(t, listener, profile, connChan)
		tcpContact.C2RequirementsMet(profile, c2Config)
	}
}

func TestTCPGetBeaconBytes(t *testing.T) {
	if tcpContact, ok := contact.CommunicationChannels["tcp"]; ok {
		listener := setupTCPListener(t, tcpContact)
		defer listener.Close()

		profile, c2Config := setupProfileAndConfig()

		connChan := make(chan net.Conn, 1)
		go handleTCPHandshake(t, listener, profile, connChan)

		tcpContact.C2RequirementsMet(profile, c2Config)

		conn := <-connChan

		beaconBytes := tcpContact.GetBeaconBytes(profile)
		if beaconBytes != nil {
			t.Errorf("Received bytes from GetBeaconBytes when there should have been none: %s", string(beaconBytes))
		}

		// Creating sample beacon message containing array of instructions
		beaconMessage := make(map[string]interface{})
		beaconMessage["messageType"] = "instruction"
		instructions := make([]map[string]interface{}, 1)
		instruction := make(map[string]interface{})
		instruction["executor"] = "sh"
		instruction["command"] = "d2hvYW1pCg=="
		instructions[0] = instruction

		marshalledInstructions, err := json.Marshal(instructions)
		if err != nil {
			t.Errorf("Error while marshalling instructions array: %s", err.Error())
		}
		encodedInstructions := base64.StdEncoding.EncodeToString(marshalledInstructions)
		beaconMessage["message"] = encodedInstructions
		beaconJdata, err := json.Marshal(beaconMessage)
		if err != nil {
			t.Errorf("Error on marshalling beaconMessage in test: %s", err.Error())
		}
		_, err = conn.Write(beaconJdata)
		if err != nil {
			t.Errorf("Error writing beaconJdata to connection: %s", err.Error())
		}

		time.Sleep(2 * time.Second)
		beaconBytes = tcpContact.GetBeaconBytes(profile)
		if beaconBytes == nil {
			t.Errorf("beaconBytes returned nil, expected return value")
			return
		}
		unmarshalData := make([]map[string]interface{}, 1)
		err = json.Unmarshal(beaconBytes, &unmarshalData)
		if err != nil {
			t.Errorf("Error on unmarshalling beaconBytes output in test: %s. Json input is: %s", err.Error(), beaconBytes)
		}

		if !reflect.DeepEqual(unmarshalData, instructions) {
			t.Errorf("Received message does not match expected value. Received: %s. Expected: %s", unmarshalData, instructions)
		}

	}
}

func TestTCPGetPayloadBytes(t *testing.T) {
	if tcpContact, ok := contact.CommunicationChannels["tcp"]; ok {
		listener := setupTCPListener(t, tcpContact)
		defer listener.Close()

		profile, c2Config := setupProfileAndConfig()

		connChan := make(chan net.Conn, 1)
		go handleTCPHandshake(t, listener, profile, connChan)

		tcpContact.C2RequirementsMet(profile, c2Config)

		conn := <-connChan

		payloadName := "test.file"
		payloadBytes := "This is test text. Hi!"

		requestFields := make(map[string]interface{})
		requestFields["messageType"] = "payloadRequest"
		requestFields["payload"] = payloadName
		requestFields["paw"] = profile["paw"]
		requestFields["platform"] = profile["platform"]

		// goroutine handles payload request from client when GetPayloadBytes is called
		go func() {
			// When GetPayloadBytes is called, the agent requests the payload from the server
			// We process the request, and then return the payload data
			buf := make([]byte, 2048)
			_, err := conn.Read(buf)
			if err != nil {
				t.Errorf("Error reading from TCP connection: %s", err.Error())
			}
			buf = bytes.Trim(buf, "\x00")

			var unmarshalData map[string]interface{}
			err = json.Unmarshal(buf, &unmarshalData)
			if err != nil {
				t.Errorf("Error unmarshalling response: %s", err.Error())
			}

			if !reflect.DeepEqual(requestFields, unmarshalData) {
				t.Errorf("Actual payload request does not match expected. Actual: %s. Expected: %s", unmarshalData, requestFields)
			}

			beaconMessage := make(map[string]interface{})
			beaconMessage["messageType"] = "payload"
			payload := make(map[string]interface{})
			payload["filename"] = payloadName
			encodedBytes := base64.StdEncoding.EncodeToString([]byte(payloadBytes))
			payload["bytes"] = encodedBytes

			marshalledPayload, err := json.Marshal(payload)
			if err != nil {
				t.Errorf("Error when marshalling payload: %s", err.Error())
			}
			beaconMessage["message"] = base64.StdEncoding.EncodeToString(marshalledPayload)

			beaconJdata, err := json.Marshal(beaconMessage)
			if err != nil {
				t.Errorf("Error when marshalling beacon: %s", err.Error())
			}

			_, err = conn.Write(beaconJdata)
			if err != nil {
				t.Errorf("Error writing beaconJdata to connection: %s", err.Error())
			}
		}()

		returnedPayloadBytes, returnedPayloadName := tcpContact.GetPayloadBytes(profile, payloadName)
		if strings.Compare(returnedPayloadName, payloadName) != 0 {
			t.Errorf("Received %s for payload name, expected %s", returnedPayloadName, payloadName)
		}
		if strings.Compare(string(returnedPayloadBytes), payloadBytes) != 0 {
			t.Errorf("Received %s for payload bytes, expected %s", returnedPayloadBytes, payloadBytes)
		}
	}
}

func TestTCPSendExecutionResults(t *testing.T) {
	if tcpContact, ok := contact.CommunicationChannels["tcp"]; ok {
		listener := setupTCPListener(t, tcpContact)
		defer listener.Close()

		profile, c2Config := setupProfileAndConfig()

		connChan := make(chan net.Conn, 1)
		go handleTCPHandshake(t, listener, profile, connChan)

		tcpContact.C2RequirementsMet(profile, c2Config)

		conn := <-connChan

		//start of executionResults test
		results := make([]interface{}, 1)
		result := make(map[string]interface{})
		result["id"] = "1234"
		result["output"] = "test output"
		results[0] = result

		requestFields := make(map[string]interface{})
		for k, v := range profile {
			requestFields[k] = v
		}
		// requestFields["paw"] = profile["paw"]
		// requestFields["contact"] = profile["contact"]
		requestFields["results"] = results

		requestJdata, err := json.Marshal(requestFields)
		if err != nil {
			t.Errorf("Error marshalling requestFields in test: %s", err.Error())
		}

		encodedRequest := base64.StdEncoding.EncodeToString(requestJdata)

		beaconMessage := make(map[string]interface{})
		beaconMessage["messageType"] = "executionResults"
		beaconMessage["results"] = encodedRequest

		// Channel to ensure test ends only after following goroutine ends
		waiter := make(chan int, 1)

		go func() {
			buf := make([]byte, 2048)
			_, err := conn.Read(buf)
			if err != nil {
				t.Errorf("Error reading from TCP connection: %s", err.Error())
			}
			buf = bytes.Trim(buf, "\x00")

			unmarshalData := make(map[string]interface{})
			err = json.Unmarshal(buf, &unmarshalData)
			if err != nil {
				t.Errorf("Error unmarshalling response: %s", err.Error())
			}

			if !reflect.DeepEqual(beaconMessage, unmarshalData) {
				t.Errorf("Actual payload request does not match expected. Actual: %s. Expected: %s", unmarshalData, beaconMessage)
			}
			waiter <- 1
		}()

		tcpContact.SendExecutionResults(profile, result)
		// wait for goroutine to finish before ending test
		<-waiter
	}
}

func TestTCPGetName(t *testing.T) {
	if tcpContact, ok := contact.CommunicationChannels["tcp"]; ok {
		returnedName := tcpContact.GetName()
		if strings.Compare(returnedName, "TCP") != 0 {
			t.Errorf("TCP contact name not returned as TCP: %s", returnedName)
		}
	}
}

func TestTCPUploadFileBytes(t *testing.T) {
	if tcpContact, ok := contact.CommunicationChannels["tcp"]; ok {
		listener := setupTCPListener(t, tcpContact)
		defer listener.Close()

		profile, c2Config := setupProfileAndConfig()

		connChan := make(chan net.Conn, 1)
		go handleTCPHandshake(t, listener, profile, connChan)

		tcpContact.C2RequirementsMet(profile, c2Config)

		conn := <-connChan

		upload := make(map[string]interface{})
		upload["filename"] = "test.file"
		upload["data"] = "This is test text. Hi!"
		uploadJdata, err := json.Marshal(upload)
		if err != nil {
			t.Errorf("[-] Error while marshalling upload data: %s", err.Error())
		}

		encodedUploadData := base64.StdEncoding.EncodeToString(uploadJdata)

		uploadRequest := make(map[string]interface{})
		uploadRequest["messageType"] = "fileUpload"
		uploadRequest["upload"] = encodedUploadData

		// Channel to ensure test ends only after following goroutine ends
		waiter := make(chan int, 1)

		go func() {
			buf := make([]byte, 2048)
			_, err := conn.Read(buf)
			if err != nil {
				t.Errorf("Error reading from TCP connection: %s", err.Error())
			}
			buf = bytes.Trim(buf, "\x00")

			unmarshalData := make(map[string]interface{})
			err = json.Unmarshal(buf, &unmarshalData)
			if err != nil {
				t.Errorf("Error unmarshalling response: %s", err.Error())
			}

			decodedUpload, err := base64.StdEncoding.DecodeString(unmarshalData["upload"].(string))
			if err != nil {
				t.Errorf("Error decoding upload data: %s", err.Error())
			}
			unmarshalDecoded := make(map[string]interface{})
			err = json.Unmarshal(decodedUpload, &unmarshalDecoded)
			if err != nil {
				t.Errorf("Error unmarshalling decoded upload: %s", err.Error())
			}
			decodedFileBytes, err := base64.StdEncoding.DecodeString(unmarshalDecoded["data"].(string))
			if err != nil {
				t.Errorf("Error decoding file bytes: %s", err.Error())
			}
			if strings.Compare(string(decodedFileBytes), upload["data"].(string)) != 0 {
				t.Errorf("Received file data does not match actual. Actual: %s. Expected: %s", string(decodedFileBytes), upload["data"].(string))
			}

			waiter <- 1
		}()

		tcpContact.UploadFileBytes(profile, upload["filename"].(string), []byte(upload["data"].(string)))

		<-waiter
	}
}

func setupTCPListener(t *testing.T, tcpContact contact.Contact) net.Listener {
	serverAddr := "127.0.0.1:7010"
	tcpContact.SetUpstreamDestAddr(serverAddr)

	// Start tcp listener for agent connection
	listener, err := net.Listen("tcp", "127.0.0.1:7010")
	if err != nil {
		t.Errorf("Error starting TCP listener: %s", err.Error())
	}
	t.Log("Started Test Listener")
	return listener
}

func setupProfileAndConfig() (map[string]interface{}, map[string]string) {
	profile := make(map[string]interface{})
	profile["paw"] = "abcdef"
	profile["testField"] = "test123"
	profile["platform"] = "testPlatform"
	profile["contact"] = "tcp"
	c2Config := make(map[string]string)
	return profile, c2Config
}

func handleTCPHandshake(t *testing.T, listener net.Listener, profile map[string]interface{}, connChan chan net.Conn) {
	conn, err := listener.Accept()
	if err != nil {
		t.Errorf("Error accepting TCP connection: %s", err.Error())
	}
	t.Log("Received connection")

	buf := make([]byte, 2048)
	_, err = conn.Read(buf)
	if err != nil {
		t.Errorf("Error reading from TCP connection: %s", err.Error())
	}
	buf = bytes.Trim(buf, "\x00")

	unmarshalData := make(map[string]interface{})
	err = json.Unmarshal(buf, &unmarshalData)
	if err != nil {
		t.Errorf("Error unmarshalling response: %s", err.Error())
	}

	if !reflect.DeepEqual(unmarshalData, profile) {
		t.Errorf("Received message does not match expected value. Received: %s. Expected: %s", unmarshalData, profile)
	}

	jdata, err := json.Marshal(profile)
	if err != nil {
		t.Errorf("Error on marshalling profile in test: %s", err.Error())
	}
	conn.Write(jdata)
	connChan <- conn
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
