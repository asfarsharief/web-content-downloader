package httpservice

import (
	"crypto/tls"
	"io"
	"net/http"
	"time"
	"web-content-downloader/pkg/logger"
)

// Get - Get function for the Rest API calls
func Get(url string) (*http.Response, error) {
	client := createHTTPClient()

	clientRequest, err := createHTTPRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(clientRequest)
}

// createHTTPClient : create http client with timeout and auth details
func createHTTPClient() http.Client {
	timeout := time.Duration(15) * time.Second
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
}

// createHTTPRequest : returns http request object for the Api calls,
// TODO Add methods for customizing header and Auth
func createHTTPRequest(requestType string, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(requestType, url, body)
	if err != nil {
		logger.Errorf("%s", "error while creating request object")
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	return request, nil
}
