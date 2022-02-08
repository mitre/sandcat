package contact

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/github"
)

const (
	GIST_NAME = "GIST"
	RANDOM_ID = "123456"
	GIVEN_PAW = "givenpaw"
	C2_KEY = "thisisadummykey"
	DUMMY_GIST_NAME = "dummygistname"
	DUMMY_GIST_DESC = "dummygistdesc"
	SUCCESS_STATUS = 200
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
	postedGist *github.Gist
	DUMMY_GIST_DATA = []byte("data123123")
)

func MockGetClient(token string) *github.Client {
	return &github.Client{}
}

func MockGetGistRandomIdentifier() string {
	return RANDOM_ID
}

func MockPostGistSuccessful(client *github.Client, ctx context.Context, gist *github.Gist) (*github.Gist, *github.Response, error) {
	respStruct := &github.Response{
		&http.Response{
			StatusCode: 200,
			Status: "200 OK",
		},
		1,
		1,
		1,
		1,
		github.Rate{
			10,
			10,
			github.Timestamp{
				time.Now(),
			},
		},
	}
	postedGist = gist
	return gist, respStruct, nil
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
		gistPoster: MockPostGistSuccessful,
	}
	return GenerateGistContactHandler(gistFuncHandles)
}

func TestGenerateGistHandler(t *testing.T) {
	want := &GIST{
		name: GIST_NAME,
		clientGetter: MockGetClient,
		randomIdGetter: MockGetGistRandomIdentifier,
		gistPoster: MockPostGistSuccessful,
	}
	generated := generateTestGistContactHandler()
	if generated.name != want.name {
		t.Errorf("got '%s' as gist handler's name; expected '%s'", generated.name, want.name)
	}
	compareFunctionAddr(t, generated.clientGetter, want.clientGetter)
	compareFunctionAddr(t, generated.randomIdGetter, want.randomIdGetter)
	compareFunctionAddr(t, generated.gistPoster, want.gistPoster)
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

func TestCreateAndPostGist(t *testing.T) {
	handler := generateTestGistContactHandler()
	postedGist = &github.Gist{}
	result := handler.createAndPostGist(DUMMY_GIST_NAME, DUMMY_GIST_DESC, DUMMY_GIST_DATA)
	if result != SUCCESS_STATUS {
		t.Errorf("got '%d' status code from posting gist; expected '%d'", result, SUCCESS_STATUS)
	}
	verifyPostedGist(t, postedGist, DUMMY_GIST_NAME, DUMMY_GIST_DESC, DUMMY_GIST_DATA)
}

func verifyPostedGist(t *testing.T, postedGist *github.Gist, wantName, wantDesc string, wantData []byte) {
	expectedDataEnc := base64.StdEncoding.EncodeToString(wantData)
	if *postedGist.Description != wantDesc {
		t.Errorf("got '%s' from posted GIST description; expected '%s'", *postedGist.Description, wantDesc)
	}
	if *postedGist.Public {
		t.Error("Posted gist must not be public.")
	}
	numFilesPosted := len(postedGist.Files)
	numFilesExpected := 1
	if numFilesPosted != numFilesExpected {
		t.Errorf("got '%d' files posted for gist; expected '%d'", numFilesPosted, numFilesExpected)
	}
	postedFile, ok := postedGist.Files[github.GistFilename(wantName)]
	if !ok {
		t.Errorf("Expected file %s to be posted, but it was not", wantName)
	} else {
		if *postedFile.Content != expectedDataEnc {
			t.Errorf("got '%s' from posted GIST content; expected '%s'", *postedFile.Content, expectedDataEnc)
		}
	}
}