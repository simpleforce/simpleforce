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

	duplicateRuleHeader = "Sforce-Duplicate-Rule-Header"
)

type Client interface {
	Query(query, nextRecordsURL string) (*QueryResult, error)

	DescribeSObject(sobj *SObject) (*SObjectMeta, error)
	CreateSObject(sobj *SObject, blacklistedFields []string, allowDuplicates bool) error
	GetSObject(sobj *SObject) error
	UpdateSObject(sobj *SObject, blacklistedFields []string) error
	UpsertSObject(sobject *SObject, idField, idValue string, blacklistedFields []string) error
	DeleteSObject(sobj *SObject) error

	DescribeGlobal() (*SObjectMeta, error)
	DownloadFile(contentVersionID string, filepath string) error
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
	TotalSize      int        `json:"totalSize"`
	Done           bool       `json:"done"`
	NextRecordsURL string     `json:"nextRecordsUrl"`
	Records        []*SObject `json:"records"`
}

// Query runs an SOQL query.
// nextRecordsURL is used for iterating paginated results.
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

	result := &QueryResult{}

	err = json.NewDecoder(res.Body).Decode(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DescribeSObject queries the metadata of an SObject using the "describe" API.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/resources_sobject_describe.htm
func (h *HTTPClient) DescribeSObject(sobj *SObject) (*SObjectMeta, error) {
	if len(sobj.Type()) == 0 {
		return nil, ErrInvalidSObject{"Type is empty"}
	}

	url := h.makeURL("sobjects/" + sobj.Type() + "/describe")

	res, err := h.request(http.MethodGet, url, nil, nil)
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

type createSObjectResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

// CreateSObject POSTs the JSON representation of the SObject to salesforce to create the entry.
// If the creation is successful, the ID of the SObject instance is updated with the ID returned.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/dome_sobject_create.htm
func (h *HTTPClient) CreateSObject(sobj *SObject, blacklistedFields []string, allowDuplicates bool) error {
	if len(sobj.Type()) == 0 {
		return ErrInvalidSObject{"Type is empty"}
	}

	// Make a copy of the incoming SObject, skipping certain metadata fields as they're not understood by salesforce.
	reqObj := sobj.makeCopy(blacklistedFields)
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}

	url := h.makeURL("sobjects/" + sobj.Type() + "/")

	var headers http.Header
	if allowDuplicates {
		headers = http.Header{}
		headers.Set(duplicateRuleHeader, "allowSave=true")
	}

	res, err := h.request(http.MethodPost, url, bytes.NewReader(reqData), headers)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var resData *createSObjectResponse

	err = json.NewDecoder(res.Body).Decode(&resData)
	if err != nil {
		return err
	}

	if !resData.Success || resData.ID == "" {
		return ErrFailure
	}

	sobj.SetID(resData.ID)

	return nil
}

// GetSObject retrieves all the data fields of an SObject.
func (h *HTTPClient) GetSObject(sobj *SObject) error {
	if len(sobj.Type()) == 0 {
		return ErrInvalidSObject{"Type is empty"}
	}

	if len(sobj.ID()) == 0 {
		return ErrInvalidSObject{"Id is empty"}
	}

	url := h.makeURL("sobjects/" + sobj.Type() + "/" + sobj.ID())

	res, err := h.request(http.MethodGet, url, nil, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(sobj)
	if err != nil {
		return err
	}

	return nil
}

// UpdateSObject updates SObject in place.
func (h *HTTPClient) UpdateSObject(sobj *SObject, blacklistedFields []string) error {
	if len(sobj.Type()) == 0 {
		return ErrInvalidSObject{"Type is empty"}
	}

	if len(sobj.ID()) == 0 {
		return ErrInvalidSObject{"Id is empty"}
	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := sobj.makeCopy(blacklistedFields)
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}

	url := h.makeURL("sobjects/" + sobj.Type() + "/" + sobj.ID())

	res, err := h.request(http.MethodPatch, url, bytes.NewReader(reqData), nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

// UpsertSObject upserts SObject.
func (h *HTTPClient) UpsertSObject(sobj *SObject, idField, idValue string, blacklistedFields []string) error {
	if len(sobj.Type()) == 0 {
		return ErrInvalidSObject{"Type is empty"}
	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := sobj.makeCopy(blacklistedFields)
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}

	url := h.makeURL("sobjects/" + sobj.Type() + "/" + idField + "/" + idValue)

	res, err := h.request(http.MethodPatch, url, bytes.NewReader(reqData), nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

// DeleteSObject deletes an SObject record.
func (h *HTTPClient) DeleteSObject(sobj *SObject) error {
	if len(sobj.Type()) == 0 {
		return ErrInvalidSObject{"Type is empty"}
	}

	if len(sobj.ID()) == 0 {
		return ErrInvalidSObject{"Id is empty"}
	}

	url := h.makeURL("sobjects/" + sobj.Type() + "/" + sobj.ID())

	_, err := h.request(http.MethodDelete, url, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// httpRequest executes an HTTP request to the salesforce server and returns the HTTP response.
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

// makeURL generates a REST API URL based on baseURL and APIVersion of the client.
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

// DescribeGlobal lists all available objects and their metadata.
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
