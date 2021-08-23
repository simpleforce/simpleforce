package simpleforce

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
func (s *SObject) Describe() *SObjectMeta {
	if s.Type() == "" || s.client() == nil {
		// Sanity check.
		return nil
	}
	url := s.client().makeURL("sobjects/" + s.Type() + "/describe")
	data, err := s.client().httpRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	var meta SObjectMeta
	err = json.Unmarshal(data, &meta)
	if err != nil {
		return nil
	}
	return &meta
}

// Get retrieves all the data fields of an SObject. If id is provided, the SObject with the provided external ID will
// be retrieved; otherwise, the existing ID of the SObject will be checked. If the SObject doesn't contain an ID field
// and id is not provided as the parameter, nil is returned.
// If query is successful, the SObject is updated in-place and exact same address is returned; otherwise, nil is
// returned if failed.
func (s *SObject) Get(id ...string) *SObject {
	if s.Type() == "" || s.client() == nil {
		// Sanity check.
		return nil
	}

	oid := s.ID()
	if len(id) > 0 {
		oid = id[0]
	}
	if oid == "" {
		log.Println(logPrefix, "object id not found.")
		return nil
	}

	url := s.client().makeURL("sobjects/" + s.Type() + "/" + oid)
	data, err := s.client().httpRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(logPrefix, "http request failed,", err)
		return nil
	}

	err = json.Unmarshal(data, s)
	if err != nil {
		log.Println(logPrefix, "json decode failed,", err)
		return nil
	}

	return s
}

// Create posts the JSON representation of the SObject to salesforce to create the entry.
// If the creation is successful, the ID of the SObject instance is updated with the ID returned. Otherwise, nil is
// returned for failures.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/dome_sobject_create.htm
func (s *SObject) Create() *SObject {
	if s.Type() == "" || s.client() == nil {
		// Sanity check.
		return nil
	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := s.makeCopy()
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		log.Println(logPrefix, "failed to convert sobject to json,", err)
		return nil
	}

	url := s.client().makeURL("sobjects/" + s.Type() + "/")
	respData, err := s.client().httpRequest(http.MethodPost, url, bytes.NewReader(reqData))
	if err != nil {
		log.Println(logPrefix, "failed to process http request,", err)
		return nil
	}

	// Use an anonymous struct to parse the result if any. This might need to be changed if the result should
	// be returned to the caller in some manner, especially if the client would like to decode the errors.
	var respVal struct {
		ID      string `json:"id"`
		Success bool   `json:"success"`
	}
	err = json.Unmarshal(respData, &respVal)
	if err != nil {
		log.Println(logPrefix, "failed to process response data,", err)
		return nil
	}

	if !respVal.Success || respVal.ID == "" {
		log.Println(logPrefix, "unsuccessful")
		return nil
	}

	s.setID(respVal.ID)
	return s
}

// Update updates SObject in place. Upon successful, same SObject is returned for chained access.
// ID is required.
func (s *SObject) Update() *SObject {
	if s.Type() == "" || s.client() == nil || s.ID() == "" {
		// Sanity check.
		return nil
	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := s.makeCopy()
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		log.Println(logPrefix, "failed to convert sobject to json,", err)
		return nil
	}

	queryBase := "sobjects/"
	if s.client().useToolingAPI {
		queryBase = "tooling/sobjects/"
	}
	url := s.client().makeURL(queryBase + s.Type() + "/" + s.ID())
	respData, err := s.client().httpRequest(http.MethodPatch, url, bytes.NewReader(reqData))
	if err != nil {
		log.Println(logPrefix, "failed to process http request,", err)
		return nil
	}
	log.Println(string(respData))

	return s
}

// Delete deletes an SObject record identified by external ID. nil is returned if the operation completes successfully;
// otherwise an error is returned
func (s *SObject) Delete(id ...string) error {
	if s.Type() == "" || s.client() == nil {
		// Sanity check
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
	log.Println(url)
	_, err := s.client().httpRequest(http.MethodDelete, url, nil)
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
	switch value.(type) {
	case string:
		return value.(string)
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
	switch client.(type) {
	case *HTTPClient:
		return client.(*HTTPClient)
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
func (s *SObject) makeCopy() map[string]interface{} {
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
	return stripped
}
