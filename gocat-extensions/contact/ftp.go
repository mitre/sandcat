package contact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"io"
	"math"
	"math/rand"
	"strconv"
	"time"

    "github.com/jlaffaye/ftp"
	"github.com/mitre/gocat/output"
)

var (
	client, errConnect := fto.Dial("127.0.0.1:2222")
)

//API communicates through FTP
type FTP struct {
	name string
	upstreamDestAddr string
}

func init() {
	CommunicationChannels["FTP"] = FTP{ name: "FTP" }
}

//GetInstructions sends a beacon and returns instructions
func (f FTP) GetBeaconBytes(profile map[string]interface{}) []byte {
	var retProfile []byte
	retBytes, heartbeat := FtpBeacon(profile)
	if heartbeat == true {
		retProfile = retBytes
	}
	return retProfile
}

//GetPayloadBytes load payload bytes from github
func (f FTP) GetPayloadBytes(profile map[string]interface{}, payloadName string) ([]byte, string) {
	var payloadBytes []byte
	var err error
	if _, ok := profile["paw"]; !ok {
		output.VerbosePrint("[!] Error obtaining payload - profile missing paw.")
		return nil, ""
	}
	data, err := DownloadPayload(profile["paw"].(string), payloadName)
	if err != nil {
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to download payload file: %s", err.Error()))
		return nil, ""
	}
	payloadBytes, err = StringToByteArray(data)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Failed to convert file to payload bytes: %s", err.Error()))
		return nil, ""
	}
	return payloadBytes, payloadName
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (f FTP) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
    config := make(map[string]string)
        if len(profile["paw"].(string)) == 0 {
        	config["paw"] = getBeaconNameIdentifier()
            return true, config
        }
    return true, config
}

//SendExecutionResults send results to the server
func (f FTP) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}){
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := [1]map[string]interface{}{result}
	profileCopy["results"] = results
	UploadFileBytes(profile, "results.txt", profileCopy)
}

func (f FTP) GetName() string {
	return f.name
}

func (f FTP) SetUpstreamDestAddr(upstreamDestAddr string) {
    f.upstreamDestAddr = upstreamDestAddr
}

func (f ftp) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	paw := profile["paw"].(string)
	newData, err := ByteArrayToString(data, uploadName)
    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[-] Failed to convert byte array to file: %s", err.Error()))
        return err
    }

	connect := UploadToServer(uploadName, newData, paw)
	if connect != true {

		return error
	}
	return nil
}

func FtpBeacon(profile map[string]interface{}) ([]byte, bool){
    paw := profile["paw"].(string)
    data, heartbeat = ProfileToString(profile)
    if heartbeat == nil{
        output.VerbosePrint("[!] Error converting profile map to String - cannot send beacon")
        return nil, false
    }

    connect := UploadToServer("Alive.txt", data, paw)
	if connect != true {
	    output.VerbosePrint("[!] Error sending beacon to FTP Server")
		return error
	}

    RemoveFile("Alive.txt")

	response, err := FileToByteArray("Response.txt")
	if err != nil {
	    output.VerbosePrint("[!] Error converting response to byte array - cannot obtain response")
		return nil, false
	}
	return response, true

}

func ProfileToString(profile map[string]interface{}) (string, error){
    profileData, err := json.Marshal(profile)
	if err != nil {
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to json marshal profile map: %s", err.Error()))
		return nil, err
	}

    jsonStr := string(profileData)
	return jsonString, nil
}

func ByteArrayToString(data []byte, fileName string) (string, error) {
    file := string(data)
    if err != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to write byte array to string: %s", err.Error()))
        return nil, err
    }
    return file, err
}

func StringToByteArray(data string) ([]byte, error){
    fileContent := []byte(data)
    return fileContent, nil
}

func UploadToServer(fileName string, data string, paw string) bool{
	err := ServerConnectUser(paw)
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server: %s", err.Error()))
        return false
    }
	err := UploadFile(filename, data)
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to upload file to FTP Server: %s", err.Error()))
        return false
    }

    return true
}

func getBeaconNameIdentifier() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}

func DownloadPayload(paw string, payloadName string) (string, error){
    err := ServerConnectUser(paw)
    if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server anonymously: %s", err.Error()))
        return err
    }
	data, err := DownloadFile(payloadname)
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to download payload file from FTP Server anonymously: %s", err.Error()))
        return err
    }

    return data, nil
}

func RemoveFile(filename string) error{
    err := os.Remove(filename)
    if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to remove file for cleanup: %s", err.Error()))
        return err
    }

    return nil
}

func UploadFile(filename string, data string) error{
    err = client.Stor(filename, data)
    if err != nil {
	    return err
    }
    err := client.Logout()
    if err != nil {
        return err
    }
    return nil
}

func DownloadFile(filename string) (string,error){
    reader, err := client.Retr(filename)
    if err != nil {
        return nil, err
    }
    defer reader.Close()

    buf, err := ioutil.ReadAll(reader)
    if err != nil {
        return nil, err
    }
    data = string(buf)

    err := client.Logout()
    if err != nil {
        return nil, err
    }
    return data, nil
}

func ServerConnectUser(paw string) error{
    //client, err := fto.Dial("127.0.0.1:2222")
    if errConnect != nil {
    return err
    }
    // TODO: Get username and pass from default.yaml
    if err := client.Login("red", "admin"); err != nil {
    return err
    }
    // TODO: Get dir from default.yaml
    if err = ftp.Cwd("/tmp/caldera"+paw); err != nil {
        return err
    }

    return nil
}

func ServerConnectAnonymous(paw string) error{
    if errConnect != nil {
    return err
    }
    // TODO: Get username and pass from default.yaml
    if err := client.Login("anonymous", "anonymous"); err != nil {
    return err
    }
    // TODO: Get dir from default.yaml
    if err = ftp.Cwd("/tmp/caldera"+paw); err != nil {
        return err
    }

    return nil
}