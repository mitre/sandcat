package contact 

//Contact defines required functions for communicating with the server
type Contact interface {
	Ping(server string) bool
	GetInstructions(profile map[string]interface{}) map[string]interface{}
	DropPayloads(payload string, server string) []string
	RunInstruction(command map[string]interface{}, profile map[string]interface{}, payloads []string)
}

//CommunicationChannels contains the contact implementations
var CommunicationChannels = map[string]Contact{
	"API": API{},
}