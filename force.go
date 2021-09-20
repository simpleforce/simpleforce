package simpleforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	DefaultAPIVersion = "v43.0"
	DefaultClientID   = "simpleforce"

	logPrefix = "[simpleforce]"
)

type Client interface {
	DescribeGlobal() (*SObjectMeta, error)
	DownloadFile(contentVersionID string, filepath string) error
	Query(query, nextRecordsURL string) (*QueryResult, error)
	SObject(typeName ...string) *SObject
}

var _ Client = (*HTTPClient)(nil)

// HTTPClient is the main instance to access salesforce.
type HTTPClient struct {
	httpClient *http.Client
	baseURL    string
	apiVersion string
}

// NewHTTPClient creates a new instance of the client.
func NewHTTPClient(httpClient *http.Client, baseURL, apiVersion string) *HTTPClient {
	// Trim "/" from the end of baseURL
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &HTTPClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiVersion: apiVersion,
	}
}

// QueryResult holds the response data from an SOQL query.
type QueryResult struct {
	TotalSize      int       `json:"totalSize"`
	Done           bool      `json:"done"`
	NextRecordsURL string    `json:"nextRecordsUrl"`
	Records        []SObject `json:"records"`
}

// Query runs an SOQL query. q could either be the SOQL string or the nextRecordsURL.
func (h *HTTPClient) Query(query, nextRecordsURL string) (*QueryResult, error) {
	var path string

	if len(nextRecordsURL) > 0 {
		path = nextRecordsURL
	} else {
		format := "/services/data/%s/query?q=%s"
		path = fmt.Sprintf(format, h.apiVersion, url.PathEscape(query))
	}

	url := fmt.Sprintf("%s%s", h.baseURL, path)

	res, err := h.request(http.MethodGet, url, nil, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result QueryResult

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	// Reference to client is needed if the object will be further used to execute queries.
	for idx := range result.Records {
		result.Records[idx].setClient(h)
	}

	return &result, nil
}

// SObject creates an SObject instance with provided type name and associate the SObject with the client.
func (h *HTTPClient) SObject(typeName ...string) *SObject {
	obj := &SObject{}
	obj.setClient(h)

	if typeName != nil {
		obj.setType(typeName[0])
	}

	return obj
}

// httpRequest executes an HTTP request to the salesforce server and returns the response data in byte buffer.
func (h *HTTPClient) request(method, url string, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if headers == nil {
		headers = http.Header{}
	}

	if len(headers.Get("Content-Type")) == 0 {
		headers.Set("Content-Type", "application/json")
	}

	req.Header = headers

	res, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		err = parseSalesforceError(res.StatusCode, body)

		res.Body = io.NopCloser(bytes.NewBuffer(body))

		return res, err
	}

	return res, nil
}

// makeURL generates a REST API URL based on baseURL, APIVersion of the client.
func (h *HTTPClient) makeURL(url string) string {
	return fmt.Sprintf("%s/services/data/%s/%s", h.baseURL, h.apiVersion, url)
}

// DownloadFile downloads a file based on the REST API path given. Saves to filePath.
func (h *HTTPClient) DownloadFile(contentVersionID string, filepath string) error {
	path := fmt.Sprintf("/services/data/%s/sobjects/ContentVersion/%s/VersionData", h.apiVersion, contentVersionID)
	url := fmt.Sprintf("%s%s", h.baseURL, path)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json; charset=UTF-8")
	headers.Set("Accept", "application/json")

	res, err := h.request(http.MethodGet, url, nil, headers)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, res.Body)

	return err
}

//Get the List of all available objects and their metadata for your organization's data
func (h *HTTPClient) DescribeGlobal() (*SObjectMeta, error) {
	path := fmt.Sprintf("/services/data/%s/sobjects", h.apiVersion)
	url := fmt.Sprintf("%s%s", h.baseURL, path)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json; charset=UTF-8")
	headers.Set("Accept", "application/json")

	res, err := h.request(http.MethodGet, url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var meta SObjectMeta

	err = json.NewDecoder(res.Body).Decode(&meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}
