package contact

const (
	ok = 200
	created = 201
)

//Contact defines required functions for communicating with the server
type Contact interface {
	GetInstructions(profile map[string]interface{}) map[string]interface{}
	DropPayloads(payload string, server string, uniqueId string, platform string) []string
	RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string)
	C2RequirementsMet(criteria map[string]string) bool
	GetPayloadBytes(payload string, server string, uniqueID string, platform string) []byte
	SendExecutionResults(profile map[string]interface{}, result map[string]interface{})
}

//CommunicationChannels contains the contact implementations
var CommunicationChannels = map[string]Contact{}