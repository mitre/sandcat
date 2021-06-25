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

    "app/contacts/contact_ftp"
	"github.com/mitre/gocat/output"
)

var (
	server = contact_ftp.FtpServer(BaseWorld)
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
	err := DownloadPayload(profile["paw"].(string), payloadName)
	if err != nil {
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to download payload file: %s", err.Error()))
		return nil, ""
	}
	payloadBytes, err = FileToByteArray(payloadName)
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
    return false, nil
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
	err := ByteArrayToFile(data, uploadName)
    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[-] Failed to convert byte array to file: %s", err.Error()))
        return err
    }

	connect := UploadToServer(uploadName, paw)
	if connect != true {

		return error
	}
	return nil
}

func FtpBeacon(profile map[string]interface{}) ([]byte, bool){
    paw := profile["paw"].(string)
    heartbeat = ProfileToFile(profile)
    if heartbeat == nil{
        output.VerbosePrint("[!] Error converting profile map to file - cannot send beacon")
        return nil, false
    }

    connect := UploadToServer("Alive.txt", paw)
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

func ProfileToFile(profile map[string]interface{}) error{
    profileData, err := json.Marshal(profile)
	if err != nil {
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to json marshal profile map: %s", err.Error()))
		return err
	}

    jsonStr := string(profileData)
	err := ioutil.WriteFile("Alive.txt", []byte(jsonStr), 0666)
    if err != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to write json string to byte array: %s", err.Error()))
        return err
    }


}

func ByteArrayToFile(data []byte, fileName string) error {
    err := ioutil.WriteFile(fileName, data, 0666)
    if err != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to write byte array to file: %s", err.Error()))
        return err
    }
    return err
}

func FileToByteArray(fileName string) ([]byte, error){
    var fileContent
    file, err := os.Open(fileName)
    if err != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to convert file content to byte array: %s", err.Error()))
        return nil, err
    }

    fileContent = []byte(string(file))
    defer file.Close()
    RemoveFile(fileName)
    return fileContent, nil
}

func UploadToServer(fileName string, paw string) bool{
	err := server.connect_anonymous()
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server anonymously: %s", err.Error()))
        return false
    }
	err := server.upload_file(fileName, paw)
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to upload file to FTP Server anonymously: %s", err.Error()))
        return false
    }

    return true
}

func getBeaconNameIdentifier() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}

func DownloadPayload(paw string, payloadName string) error{
    err := server.connect_anonymous()
    if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server anonymously: %s", err.Error()))
        return err
    }
	err := server.download_file(fileName, paw)
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to download payload file from FTP Server anonymously: %s", err.Error()))
        return err
    }

    return nil
}

func RemoveFile(filename string) error{
    err := os.Remove(filename)
    if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to remove file for cleanup: %s", err.Error()))
        return err
    }

    return nil
}