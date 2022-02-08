package contact

import (
	"fmt"
	"testing"

	"github.com/google/go-github/github"
)

const (
	GIST_NAME = "GIST"
)

func MockGetClient(token string) *github.Client {
	return &github.Client{}
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
	}
	return GenerateGistContactHandler(gistFuncHandles)
}

func TestGenerateGistHandler(t *testing.T) {
	want := &GIST{
		name: GIST_NAME,
		clientGetter: MockGetClient,
	}
	generated := generateTestGistContactHandler()
	if generated.name != want.name {
		t.Errorf("got '%s' as gist handler's name; expected '%s'", generated.name, want.name)
	}
	compareFunctionAddr(t, generated.clientGetter, want.clientGetter)
}