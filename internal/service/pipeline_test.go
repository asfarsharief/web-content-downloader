package service_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"web-content-downloader/internal/service"
	"web-content-downloader/pkg/constants"

	"github.com/stretchr/testify/assert"
)

func TestTriggerPipeline(t *testing.T) {
	constants.MaxWorkers = 1
	service.OpenFile = func(name string) (*os.File, error) {
		content := []byte("Urls\ndummyUrl1\ndummyurl2\ndummyurl3\n")

		tmpFile, err := os.CreateTemp("", "mock_file")
		if err != nil {
			return nil, err
		}

		_, err = tmpFile.Write(content)
		if err != nil {
			return nil, err
		}
		_, err = tmpFile.Seek(0, 0)
		if err != nil {
			return nil, err
		}

		defer os.Remove(tmpFile.Name())

		return tmpFile, nil
	}
	service.Get = func(url string) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"success": true}`))),
		}, nil
	}

	service.BasePath = "."
	pipeline := service.NewPipelineStruct("dummy")
	pipeline.TriggerPipeline("dummyPath")
	name, ok := checkIfFolderIsPresent("dummy")
	assert.True(t, ok)
	fmt.Println(os.RemoveAll("./" + name))
}

func checkIfFolderIsPresent(prefix string) (string, bool) {
	entries, err := os.ReadDir("./")
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			return entry.Name(), true
		}
	}

	return "", false
}
