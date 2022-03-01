package contact

import (
	"net"
)

type TCP struct {
	conn       net.Conn
	name       string
	serverAddr string
}

func init() {
	CommunicationChannels["tcp"] = TCP{}
}

func (t *TCP) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {

}

func (t *TCP) GetBeaconBytes(profile map[string]interface{}) []byte {

}

func (t *TCP) GetPayloadBytes(profile map[string]interface{}, payload string) ([]byte, string) {

}

func (t *TCP) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {

}

func (t *TCP) GetName() string {
	return t.name
}

func (t *TCP) SetUpstreamDestAddr(upstreamDestAddr string) {

}

func (t *TCP) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {

}

func (t *TCP) SupportsContinuous() bool {
	return true
}
