package simpleforce

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// ExecuteAnonymousResult is returned by ExecuteAnonymous function
type ExecuteAnonymousResult struct {
	Line                int         `json:"line"`
	Column              int         `json:"column"`
	Compiled            bool        `json:"compiled"`
	Success             bool        `json:"success"`
	CompileProblem      interface{} `json:"compileProblem"`
	ExceptionStackTrace interface{} `json:"exceptionStackTrace"`
	ExceptionMessage    interface{} `json:"exceptionMessage"`
}

// Tooling is called to specify Tooling API, e.g. client.Tooling().Query(q)
func (h *HTTPClient) Tooling() *HTTPClient {
	h.useToolingAPI = true
	return h
}

func (h *HTTPClient) UnTooling() {
	h.useToolingAPI = false
}

// ExecuteAnonymous executes a body of Apex code
func (h *HTTPClient) ExecuteAnonymous(apexBody string) (*ExecuteAnonymousResult, error) {
	if !h.isLoggedIn() {
		return nil, ErrAuthentication
	}

	// Create the endpoint
	formatString := "%s/services/data/v%s/tooling/executeAnonymous/?anonymousBody=%s"
	baseURL := h.instanceURL
	endpoint := fmt.Sprintf(formatString, baseURL, h.apiVersion, url.QueryEscape(apexBody))

	data, err := h.httpRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		log.Println(logPrefix, "HTTP GET request failed:", endpoint)
		return nil, err
	}

	var result ExecuteAnonymousResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
