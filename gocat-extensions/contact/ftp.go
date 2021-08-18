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
    BEACON = "Alive.txt"
    PAYLOAD = "Payload.txt"
    RESPONSE = "Response.txt"
)

//API communicates through FTP
type FTP struct {
    name string
    ipAddress string
    client *ftp.ServerConn
    user string
    pword string
    directory string
    beacon string
    payload string
    response string
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
    payloadReqDict, err := CreatePayloadRequest(profile, payloadName)
    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[!] Error creating payload dictionary: %s", err.Error()))
        return nil, ""
    }
    err = f.UploadFileBytes(profile, f.payload, payloadReqDict)
    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[-] Failed to download payload file: %s", err.Error()))
        return nil, ""
    }
    data, err := f.DownloadFile(payloadName)
    if err != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to download file from FTP Server: %s", err.Error()))
        return nil, ""
    }
    f.client.Delete(payloadName)
    return data, payloadName
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (f *FTP) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
    config := make(map[string]string)
    if len(profile["paw"].(string)) == 0 {
        config["paw"] = getRandomId()
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
        err = f.UploadFileBytes(profile, f.beacon, data)
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
    f.directory = "/" + DIRECTORY
    f.beacon = BEACON
    f.payload = PAYLOAD
    f.response = RESPONSE

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
    if uniqueFileName != f.beacon && uniqueFileName != f.payload{
        uploadId := getRandomId()
        uniqueFileName = uploadName + "-" + uploadId
    }

    errConn := f.ServerSetDir(paw)
    if errConn != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to connect to FTP Server: %s", errConn.Error()))
        return errConn
    }

    errUpload := f.UploadFile(uniqueFileName, data)
    if errUpload != nil {
        return errUpload
    }

    return nil
}

func CreatePayloadRequest(profile map[string]interface{}, payloadName string) ([]byte, error) {
    platform := profile["platform"]
    paw := profile["paw"]
    if platform == nil && paw == nil {
        output.VerbosePrint("[!] Error obtaining payload - profile missing paw and/or platform.")
        return nil, errors.New("profile does not contain platform and/or paw")
    }

    payloadReqDict := map[string]string{
                                            "file": payloadName,
                                            "platform": platform.(string),
                                            "paw": paw.(string),
                                        }
    data, err := json.Marshal(payloadReqDict)
    if err != nil {
        output.VerbosePrint(fmt.Sprintf("[-] Failed to json marshal payload request map: %s", err.Error()))
        return nil, err
    }

    return data, nil
}

//Connect to ftp server with username and password
func (f *FTP) ServerSetDir(paw string) error {
    if err := f.client.ChangeDir(f.directory + "/" + paw); err != nil {
        if err := f.client.MakeDir(f.directory + "/" + paw); err != nil{
            return err
        }
        f.client.ChangeDir(f.directory + "/" + paw)
    }

    return nil
}

//Controls process to send beacon to server
func (f *FTP) FtpBeacon(profile map[string]interface{}) ([]byte, bool) {
    data, heartbeat := json.Marshal(profile)
    if heartbeat != nil{
        output.VerbosePrint("[!] Error converting profile map to String - cannot send beacon")
        return nil, false
    }

    connectErr := f.UploadFileBytes(profile, f.beacon, data)
    if connectErr != nil {
        output.VerbosePrint("[!] Error sending beacon to FTP Server")
        return nil, false
    }

    data, err := f.DownloadFile(f.response)
    if err != nil{
        output.VerbosePrint(fmt.Sprintf("[-] Failed to download file from FTP Server: %s", err.Error()))
        return nil, false
    }
    return data, true

}

//Upload file to server
func (f *FTP) UploadFile(filename string, data []byte) error {
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
    if filename == f.response{
        f.client.Delete(filename)
        f.client.Delete(f.beacon)
    }
    if filename != f.beacon && filename != f.response{
        f.client.Delete(f.payload)
    }
    return data, nil
}

//If no paw, create one
func getRandomId() string {
    rand.Seed(time.Now().UnixNano())
    return strconv.Itoa(rand.Int())
}