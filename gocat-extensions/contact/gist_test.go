package contact

import (
	"fmt"
	"testing"

	"github.com/google/go-github/github"
)

const (
	GIST_NAME = "GIST"
	RANDOM_ID = "123456"
	GIVEN_PAW = "givenpaw"
	C2_KEY = "thisisadummykey"
)

var (
	testGistProfileNoPaw = map[string]interface{}{
		"paw": "",
	}
	testGistProfile = map[string]interface{} {
		"paw": GIVEN_PAW,
	}
	testGistCriteria = map[string]string{
		"c2Key": C2_KEY,
	}
)

func MockGetClient(token string) *github.Client {
	return &github.Client{}
}

func MockGetGistRandomIdentifier() string {
	return RANDOM_ID
}

func compareFunctionAddr(t *testing.T, outputFunc interface{}, wantFunc interface{}) {
	outputFuncAddr := fmt.Sprintf("%v", outputFunc)
	wantFuncAddr := fmt.Sprintf("%v", wantFunc)
	if outputFuncAddr != wantFuncAddr {
		t.Errorf("got '%s' func address; expected '%s'", outputFuncAddr, wantFuncAddr)
	}
}

func generateTestGistContactHandler() *GIST {
	gistFuncHandles := &GistFunctionHandles{
		clientGetter: MockGetClient,
		randomIdGetter: MockGetGistRandomIdentifier,
	}
	return GenerateGistContactHandler(gistFuncHandles)
}

func TestGenerateGistHandler(t *testing.T) {
	want := &GIST{
		name: GIST_NAME,
		clientGetter: MockGetClient,
		randomIdGetter: MockGetGistRandomIdentifier,
	}
	generated := generateTestGistContactHandler()
	if generated.name != want.name {
		t.Errorf("got '%s' as gist handler's name; expected '%s'", generated.name, want.name)
	}
	compareFunctionAddr(t, generated.clientGetter, want.clientGetter)
	compareFunctionAddr(t, generated.randomIdGetter, want.randomIdGetter)
}

func TestGistC2ReqsMetNoKey(t *testing.T) {
	handler := generateTestGistContactHandler()
	reqsMet, result := handler.C2RequirementsMet(testGistProfile, map[string]string{})
	if reqsMet {
		t.Error("C2 requirements cannot not be met if no C2 key was provided")
	}
	if result != nil {
		t.Errorf("got '%v' from C2RequirementsMet without c2 key; expected nil", result)
	}
}

func TestGistC2ReqsMetNoPaw(t *testing.T) {
	handler := generateTestGistContactHandler()
	reqsMet, result := handler.C2RequirementsMet(testGistProfileNoPaw, testGistCriteria)
	if !reqsMet {
		t.Error("Expected C2RequirementsMet to return true. Got false instead.")
	}
	if result == nil {
		t.Errorf("got nil from C2RequirementsMet; expected %v", result)
	}
	if result["paw"] != RANDOM_ID {
		t.Errorf("got '%s' as handler's paw; expected '%s'", result["paw"], RANDOM_ID)
	}
	if handler.token != C2_KEY {
		t.Errorf("got '%s' as handler's token; expected '%s'", handler.token, C2_KEY)
	}
}

func TestGistC2ReqsMetGivenPaw(t *testing.T) {
	handler := generateTestGistContactHandler()
	reqsMet, result := handler.C2RequirementsMet(testGistProfile, testGistCriteria)
	if !reqsMet {
		t.Error("Expected C2RequirementsMet to return true. Got false instead.")
	}
	if result == nil {
		t.Errorf("got nil from C2RequirementsMet; expected %v", result)
	}
	if len(result) != 0 {
		t.Errorf("got '%v' as result. expected empty map", result)
	}
	if handler.token != C2_KEY {
		t.Errorf("got '%s' as handler's token; expected '%s'", handler.token, C2_KEY)
	}
}

func TestGistGetName(t *testing.T) {
	handler := generateTestGistContactHandler()
	name := handler.GetName()
	if name != GIST_NAME {
		t.Errorf("got '%s' from GetName; expected '%s'", name, GIST_NAME)
	}
}