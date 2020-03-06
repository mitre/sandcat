package contact

import (
	"sync"
)

const (
	ok = 200
	created = 201
)

//Contact defines required functions for communicating with the server
type Contact interface {
	GetInstructions(profile map[string]interface{}) map[string]interface{}
	GetPayloadBytes(payload string, server string, uniqueID string, platform string, writeToDisk bool) (string, []byte)
	RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string, wg *sync.WaitGroup)
	C2RequirementsMet(criteria map[string]string) bool
	SendExecutionResults(profile map[string]interface{}, result map[string]interface{})
}

//CommunicationChannels contains the contact implementations
var CommunicationChannels = map[string]Contact{}