package simpleforce

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	DefaultAPIVersion = "43.0"
	DefaultClientID   = "simpleforce"
	DefaultURL        = "https://login.salesforce.com"

	logPrefix = "[simpleforce]"
)

type Client interface {
	DescribeGlobal() (*SObjectMeta, error)
	DownloadFile(contentVersionID string, filepath string) error
	ExecuteAnonymous(apexBody string) (*ExecuteAnonymousResult, error)
	GetLoc() (loc string)
	GetSid() (sid string)
	Login(username, password, token string) error
	Query(q string) (*QueryResult, error)
	SetSidLoc(sid string, loc string)
	SObject(typeName ...string) *SObject
	Tooling() *HTTPClient
	UnTooling()
}

var _ Client = (*HTTPClient)(nil)

// HTTPClient is the main instance to access salesforce.
type HTTPClient struct {
	sessionID string
	user      struct {
		id       string
		name     string
		fullName string
		email    string
	}
	clientID      string
	apiVersion    string
	baseURL       string
	instanceURL   string
	useToolingAPI bool
	httpClient    *http.Client
}

// NewHTTPClient creates a new instance of the client.
func NewHTTPClient(httpClient *http.Client, url, clientID, apiVersion string) *HTTPClient {
	client := &HTTPClient{
		apiVersion: apiVersion,
		baseURL:    url,
		clientID:   clientID,
		httpClient: httpClient,
	}

	// Append "/" to the end of baseURL if not yet.
	if !strings.HasSuffix(client.baseURL, "/") {
		client.baseURL = client.baseURL + "/"
	}
	return client
}

// QueryResult holds the response data from an SOQL query.
type QueryResult struct {
	TotalSize      int       `json:"totalSize"`
	Done           bool      `json:"done"`
	NextRecordsURL string    `json:"nextRecordsUrl"`
	Records        []SObject `json:"records"`
}

// Expose sid to save in admin settings
func (h *HTTPClient) GetSid() (sid string) {
	return h.sessionID
}

//Expose Loc to save in admin settings
func (h *HTTPClient) GetLoc() (loc string) {
	return h.instanceURL
}

// Set SID and Loc as a means to log in without Login()
func (h *HTTPClient) SetSidLoc(sid string, loc string) {
	h.sessionID = sid
	h.instanceURL = loc
}

// Query runs an SOQL query. q could either be the SOQL string or the nextRecordsURL.
func (h *HTTPClient) Query(q string) (*QueryResult, error) {
	if !h.isLoggedIn() {
		return nil, ErrAuthentication
	}

	var u string
	if strings.HasPrefix(q, "/services/data") {
		// q is nextRecordsURL.
		u = fmt.Sprintf("%s%s", h.instanceURL, q)
	} else {
		// q is SOQL.
		formatString := "%s/services/data/v%s/query?q=%s"
		baseURL := h.instanceURL
		if h.useToolingAPI {
			formatString = strings.Replace(formatString, "query", "tooling/query", -1)
		}
		u = fmt.Sprintf(formatString, baseURL, h.apiVersion, url.PathEscape(q))
	}

	data, err := h.httpRequest(http.MethodGet, u, nil)
	if err != nil {
		log.Println(logPrefix, "HTTP GET request failed:", u)
		return nil, err
	}

	var result QueryResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	// Reference to client is needed if the object will be further used to do online queries.
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

// isLoggedIn returns if the login to salesforce is successful.
func (client *HTTPClient) isLoggedIn() bool {
	return client.sessionID != ""
}

// Login signs into salesforce using password. token is optional if trusted IP is configured.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/intro_understanding_username_password_oauth_flow.htm
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api.meta/api/sforce_api_calls_login.htm
func (h *HTTPClient) Login(username, password, token string) error {
	// Use the SOAP interface to acquire session ID with username, password, and token.
	// Do not use REST interface here as REST interface seems to have strong checking against client_id, while the SOAP
	// interface allows a non-exist placeholder client_id to be used.
	soapBody := `<?xml version="1.0" encoding="utf-8" ?>
        <env:Envelope
                xmlns:xsd="http://www.w3.org/2001/XMLSchema"
                xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                xmlns:env="http://schemas.xmlsoap.org/soap/envelope/"
                xmlns:urn="urn:partner.soap.sforce.com">
            <env:Header>
                <urn:CallOptions>
                    <urn:client>%s</urn:client>
                    <urn:defaultNamespace>sf</urn:defaultNamespace>
                </urn:CallOptions>
            </env:Header>
            <env:Body>
                <n1:login xmlns:n1="urn:partner.soap.sforce.com">
                    <n1:username>%s</n1:username>
                    <n1:password>%s%s</n1:password>
                </n1:login>
            </env:Body>
        </env:Envelope>`
	soapBody = fmt.Sprintf(soapBody, h.clientID, username, html.EscapeString(password), token)

	url := fmt.Sprintf("%s/services/Soap/u/%s", h.baseURL, h.apiVersion)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(soapBody))
	if err != nil {
		log.Println(logPrefix, "error occurred creating request,", err)
		return err
	}

	req.Header.Add("Content-Type", "text/xml")
	req.Header.Add("charset", "UTF-8")
	req.Header.Add("SOAPAction", "login")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Println(logPrefix, "error occurred submitting request,", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println(logPrefix, "request failed,", resp.StatusCode)
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		newStr := buf.String()
		log.Println(logPrefix, "Failed resp.body: ", newStr)
		theError := ParseSalesforceError(resp.StatusCode, buf.Bytes())

		return theError
	}

	respData, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println(logPrefix, "error occurred reading response data,", err)
	}

	var loginResponse struct {
		XMLName      xml.Name `xml:"Envelope"`
		ServerURL    string   `xml:"Body>loginResponse>result>serverUrl"`
		SessionID    string   `xml:"Body>loginResponse>result>sessionId"`
		UserID       string   `xml:"Body>loginResponse>result>userId"`
		UserEmail    string   `xml:"Body>loginResponse>result>userInfo>userEmail"`
		UserFullName string   `xml:"Body>loginResponse>result>userInfo>userFullName"`
		UserName     string   `xml:"Body>loginResponse>result>userInfo>userName"`
	}

	err = xml.Unmarshal(respData, &loginResponse)
	if err != nil {
		log.Println(logPrefix, "error occurred parsing login response,", err)
		return err
	}

	// Now we should all be good and the sessionID can be used to talk to salesforce further.
	h.sessionID = loginResponse.SessionID
	h.instanceURL = parseHost(loginResponse.ServerURL)
	h.user.id = loginResponse.UserID
	h.user.name = loginResponse.UserName
	h.user.email = loginResponse.UserEmail
	h.user.fullName = loginResponse.UserFullName

	log.Println(logPrefix, "User", h.user.name, "authenticated.")
	return nil
}

// httpRequest executes an HTTP request to the salesforce server and returns the response data in byte buffer.
func (h *HTTPClient) httpRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", h.sessionID))
	req.Header.Add("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Println(logPrefix, "request failed,", resp.StatusCode)
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		newStr := buf.String()
		theError := ParseSalesforceError(resp.StatusCode, buf.Bytes())
		log.Println(logPrefix, "Failed resp.body: ", newStr)

		return nil, theError
	}

	return ioutil.ReadAll(resp.Body)
}

// makeURL generates a REST API URL based on baseURL, APIVersion of the client.
func (h *HTTPClient) makeURL(req string) string {
	h.apiVersion = strings.Replace(h.apiVersion, "v", "", -1)
	retURL := fmt.Sprintf("%s/services/data/v%s/%s", h.instanceURL, h.apiVersion, req)
	return retURL
}

// DownloadFile downloads a file based on the REST API path given. Saves to filePath.
func (h *HTTPClient) DownloadFile(contentVersionID string, filepath string) error {
	apiPath := fmt.Sprintf("/services/data/v%s/sobjects/ContentVersion/%s/VersionData", h.apiVersion, contentVersionID)
	baseURL := strings.TrimRight(h.baseURL, "/")
	url := fmt.Sprintf("%s%s", baseURL, apiPath)

	// Get the data
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+h.sessionID)

	// resp, err := http.Get(url)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func parseHost(input string) string {
	parsed, err := url.Parse(input)
	if err == nil {
		return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	}
	return "Failed to parse URL input"
}

//Get the List of all available objects and their metadata for your organization's data
func (h *HTTPClient) DescribeGlobal() (*SObjectMeta, error) {
	apiPath := fmt.Sprintf("/services/data/v%s/sobjects", h.apiVersion)
	baseURL := strings.TrimRight(h.baseURL, "/")
	url := fmt.Sprintf("%s%s", baseURL, apiPath) // Get the objects

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+h.sessionID)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var meta SObjectMeta

	respData, err := ioutil.ReadAll(resp.Body)
	log.Println(logPrefix, fmt.Sprintf("status code %d", resp.StatusCode))
	if err != nil {
		log.Println(logPrefix, "error while reading all body")
	}

	err = json.Unmarshal(respData, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
