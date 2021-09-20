package simpleforce

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

const (
	sobjectClientKey     = "__client__" // private attribute added to locate client instance.
	sobjectAttributesKey = "attributes" // points to the attributes structure which should be common to all SObjects.
	sobjectIDKey         = "Id"
)

var (
	// When updating existing records, certain fields are read only and needs to be removed before submitted to Salesforce.
	// Following list of fields are extracted from INVALID_FIELD_FOR_INSERT_UPDATE error message.
	blacklistedUpdateFields = []string{
		"LastModifiedDate",
		"LastReferencedDate",
		"IsClosed",
		"ContactPhone",
		"CreatedById",
		"CaseNumber",
		"ContactFax",
		"ContactMobile",
		"IsDeleted",
		"LastViewedDate",
		"SystemModstamp",
		"CreatedDate",
		"ContactEmail",
		"ClosedDate",
		"LastModifiedById",
	}
)

// SObject describes an instance of SObject.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/resources_sobject_basic_info.htm
type SObject map[string]interface{}

// SObjectMeta describes the metadata returned by describing the object.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/resources_sobject_describe.htm
type SObjectMeta map[string]interface{}

// SObjectAttributes describes the basic attributes (type and url) of an SObject.
type SObjectAttributes struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Describe queries the metadata of an SObject using the "describe" API.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/resources_sobject_describe.htm
func (s *SObject) Describe() (*SObjectMeta, error) {
	if s.Type() == "" || s.client() == nil {
		// Sanity check.
		return nil, ErrFailure
	}

	url := s.client().makeURL("sobjects/" + s.Type() + "/describe")

	res, err := s.client().request(http.MethodGet, url, nil, nil)
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

// Get retrieves all the data fields of an SObject. If id is provided, the SObject with the provided external ID will
// be retrieved; otherwise, the existing ID of the SObject will be checked. If the SObject doesn't contain an ID field
// and id is not provided as the parameter, nil is returned.
// If query is successful, the SObject is updated in-place and exact same address is returned; otherwise, nil is
// returned if failed.
func (s *SObject) Get(id ...string) (*SObject, error) {
	if s.Type() == "" || s.client() == nil {
		// Sanity check.
		return nil, ErrFailure
	}

	oid := s.ID()
	if len(id) > 0 {
		oid = id[0]
	}

	if oid == "" {
		return nil, errors.New("object id not found.")
	}

	url := s.client().makeURL("sobjects/" + s.Type() + "/" + oid)

	res, err := s.client().request(http.MethodGet, url, nil, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

type createSObjectResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

// Create posts the JSON representation of the SObject to salesforce to create the entry.
// If the creation is successful, the ID of the SObject instance is updated with the ID returned. Otherwise, nil is
// returned for failures.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/dome_sobject_create.htm
func (s *SObject) Create(blacklistedFields []string) (*SObject, error) {
	if s.Type() == "" || s.client() == nil {
		// Sanity check.
		return nil, ErrFailure
	}
	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := s.makeCopy(blacklistedFields)
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		return nil, err
	}

	url := s.client().makeURL("sobjects/" + s.Type() + "/")

	res, err := s.client().request(http.MethodPost, url, bytes.NewReader(reqData), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var resData *createSObjectResponse

	err = json.NewDecoder(res.Body).Decode(&resData)
	if err != nil {
		return nil, err
	}

	if !resData.Success || resData.ID == "" {
		return nil, ErrFailure
	}

	s.setID(resData.ID)

	return s, nil
}

// Update updates SObject in place.
// ID is required.
func (s *SObject) Update(blacklistedFields []string) error {
	if s.Type() == "" || s.client() == nil || s.ID() == "" {
		return ErrFailure

	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := s.makeCopy(blacklistedFields)
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}

	url := s.client().makeURL("sobjects/" + s.Type() + "/" + s.ID())

	res, err := s.client().request(http.MethodPatch, url, bytes.NewReader(reqData), nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

// Delete deletes an SObject record identified by external ID. nil is returned if the operation completes successfully;
// otherwise an error is returned
func (s *SObject) Delete(id ...string) error {
	if s.Type() == "" || s.client() == nil {
		return ErrFailure
	}

	oid := s.ID()
	if id != nil {
		oid = id[0]
	}
	if oid == "" {
		return ErrFailure
	}

	url := s.client().makeURL("sobjects/" + s.Type() + "/" + s.ID())

	_, err := s.client().request(http.MethodDelete, url, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// Type returns the type, or sometimes referred to as name, of an SObject.
func (s *SObject) Type() string {
	attributes := s.AttributesField()
	if attributes == nil {
		return ""
	}
	return attributes.Type
}

// ID returns the external ID of the SObject.
func (obj *SObject) ID() string {
	return obj.StringField(sobjectIDKey)
}

// StringField accesses a field in the SObject as string. Empty string is returned if the field doesn't exist.
func (s *SObject) StringField(key string) string {
	value := s.InterfaceField(key)

	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// SObjectField accesses a field in the SObject as another SObject. This is only applicable if the field is an external
// ID to another object. The typeName of the SObject must be provided. <nil> is returned if the field is empty.
func (s *SObject) SObjectField(typeName, key string) *SObject {
	// First check if there's an associated ID directly.
	oid := s.StringField(key)
	if oid != "" {
		object := &SObject{}
		object.setClient(s.client())
		object.setType(typeName)
		object.setID(oid)

		return object
	}

	// Secondly, check if this could be a linked object, which doesn't have an ID but has the attributes.
	linkedObjRaw := s.InterfaceField(key)
	linkedObjMapper, ok := linkedObjRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	attrs, ok := linkedObjMapper[sobjectAttributesKey].(map[string]interface{})
	if !ok {
		return nil
	}

	// Reusing typeName here, which is ok
	typeName, ok = attrs["type"].(string)
	url, ok := attrs["url"].(string)
	if typeName == "" || url == "" {
		return nil
	}

	// Both type and url exist in attributes, this is a linked object!
	// Get the ID from URL.
	rIndex := strings.LastIndex(url, "/")
	if rIndex == -1 || rIndex+1 == len(url) {
		// hmm... this shouldn't happen, unless the URL is hand crafted.
		log.Println(logPrefix, "invalid url,", url)
		return nil
	}

	oid = url[rIndex+1:]

	object := s.client().SObject(typeName)
	object.setID(oid)

	for key, val := range linkedObjMapper {
		object.Set(key, val)
	}

	return object
}

// InterfaceField accesses a field in the SObject as raw interface. This allows access to any type of fields.
func (s *SObject) InterfaceField(key string) interface{} {
	return (*s)[key]
}

// AttributesField returns a read-only copy of the attributes field of an SObject.
func (s *SObject) AttributesField() *SObjectAttributes {
	attributes := s.InterfaceField(sobjectAttributesKey)

	switch attributes.(type) {
	case SObjectAttributes:
		// Use a temporary variable to copy the value of attributes and return the address of the temp value.
		attrs := (attributes).(SObjectAttributes)
		return &attrs
	case map[string]interface{}:
		// Can't convert attributes to concrete type; decode interface.
		mapper := attributes.(map[string]interface{})
		attrs := &SObjectAttributes{}

		if mapper["type"] != nil {
			attrs.Type = mapper["type"].(string)
		}

		if mapper["url"] != nil {
			attrs.URL = mapper["url"].(string)
		}

		return attrs
	default:
		return nil
	}
}

// Set indexes value into SObject instance with provided key. The same SObject pointer is returned to allow
// chained access.
func (s *SObject) Set(key string, value interface{}) *SObject {
	(*s)[key] = value
	return s
}

// client returns the associated Client with the SObject.
func (s *SObject) client() *HTTPClient {
	client := s.InterfaceField(sobjectClientKey)

	switch c := client.(type) {
	case *HTTPClient:
		return c
	default:
		return nil
	}
}

// setClient sets the associated Client with the SObject.
func (s *SObject) setClient(client *HTTPClient) {
	(*s)[sobjectClientKey] = client
}

// setType sets the type, or name for the SObject.
func (s *SObject) setType(typeName string) {
	attributes := s.InterfaceField(sobjectAttributesKey)

	switch attributes.(type) {
	case SObjectAttributes:
		attrs := s.AttributesField()
		attrs.Type = typeName
		(*s)[sobjectAttributesKey] = *attrs
	default:
		(*s)[sobjectAttributesKey] = SObjectAttributes{
			Type: typeName,
		}
	}
}

// setID sets the external ID for the SObject.
func (s *SObject) setID(id string) {
	(*s)[sobjectIDKey] = id
}

// makeCopy copies the fields of an SObject to a new map without metadata fields.
func (s *SObject) makeCopy(blacklistedFields []string) map[string]interface{} {
	stripped := make(map[string]interface{})

	for key, val := range *s {
		if key == sobjectClientKey ||
			key == sobjectAttributesKey ||
			key == sobjectIDKey {
			continue
		}
		stripped[key] = val
	}

	for _, key := range blacklistedUpdateFields {
		delete(stripped, key)
	}

	for _, key := range blacklistedFields {
		delete(stripped, key)
	}

	return stripped
}
