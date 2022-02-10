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
	sobjectClientKey              = "__client__" // private attribute added to locate client instance.
	sobjectAttributesKey          = "attributes" // points to the attributes structure which should be common to all SObjects.
	sobjectIDKey                  = "Id"
	sobjectExternalIDFieldNameKey = "ExternalIDField"
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
func (obj *SObject) Describe() *SObjectMeta {
	if obj.Type() == "" || obj.client() == nil {
		// Sanity check.
		return nil
	}
	url := obj.client().makeURL("sobjects/" + obj.Type() + "/describe")
	data, err := obj.client().httpRequest(http.MethodGet, url, nil)
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
func (obj *SObject) Get(id ...string) *SObject {
	if obj.Type() == "" || obj.client() == nil {
		// Sanity check.
		return nil
	}

	oid := obj.ID()
	if len(id) > 0 {
		oid = id[0]
	}
	if oid == "" {
		log.Println(logPrefix, "object id not found.")
		return nil
	}

	url := obj.client().makeURL("sobjects/" + obj.Type() + "/" + oid)
	data, err := obj.client().httpRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(logPrefix, "http request failed,", err)
		return nil
	}

	err = json.Unmarshal(data, obj)
	if err != nil {
		log.Println(logPrefix, "json decode failed,", err)
		return nil
	}

	return obj
}

// Create posts the JSON representation of the SObject to salesforce to create the entry.
// If the creation is successful, the ID of the SObject instance is updated with the ID returned. Otherwise, nil is
// returned for failures.
// Ref: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/dome_sobject_create.htm
func (obj *SObject) Create() *SObject {
	if obj.Type() == "" || obj.client() == nil {
		// Sanity check.
		return nil
	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := obj.makeCopy()
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		log.Println(logPrefix, "failed to convert sobject to json,", err)
		return nil
	}

	url := obj.client().makeURL("sobjects/" + obj.Type() + "/")
	respData, err := obj.client().httpRequest(http.MethodPost, url, bytes.NewReader(reqData))
	if err != nil {
		log.Println(logPrefix, "failed to process http request,", err)
		return nil
	}

	err = obj.setIDFromResponseData(respData)
	if err != nil {
		log.Println(logPrefix, "failed to parse response,", err)
		return nil
	}

	return obj
}

// Update updates SObject in place. Upon successful, same SObject is returned for chained access.
// ID is required.
func (obj *SObject) Update() *SObject {
	if obj.Type() == "" || obj.client() == nil || obj.ID() == "" {
		// Sanity check.
		return nil
	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := obj.makeCopy()
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		log.Println(logPrefix, "failed to convert sobject to json,", err)
		return nil
	}

	queryBase := "sobjects/"
	if obj.client().useToolingAPI {
		queryBase = "tooling/sobjects/"
	}
	url := obj.client().makeURL(queryBase + obj.Type() + "/" + obj.ID())
	respData, err := obj.client().httpRequest(http.MethodPatch, url, bytes.NewReader(reqData))
	if err != nil {
		log.Println(logPrefix, "failed to process http request,", err)
		return nil
	}
	log.Println(string(respData))

	return obj
}

// Upsert creates SObject or updates existing SObject in place. Upon successful upsert, same SObject is returned for chained access.
// ID, ExternalIDField and Type are required. ID is the value of the external ID in this case.
func (obj *SObject) Upsert() *SObject {
	log.Println(logPrefix, "ExternalID:", obj.ExternalID())
	log.Println(logPrefix, "ExternalIDField:", obj.ExternalIDFieldName())
	if obj.Type() == "" || obj.client() == nil || obj.ExternalIDFieldName() == "" ||
		obj.ExternalID() == "" {
		// Sanity check.
		log.Println(logPrefix, "required fields are missing")
		return nil
	}

	// Make a copy of the incoming SObject, but skip certain metadata fields as they're not understood by salesforce.
	reqObj := obj.makeCopy()
	reqData, err := json.Marshal(reqObj)
	if err != nil {
		log.Println(logPrefix, "failed to convert sobject to json,", err)
		return nil
	}

	queryBase := "sobjects/"
	if obj.client().useToolingAPI {
		queryBase = "tooling/sobjects/"
	}
	url := obj.client().
		makeURL(queryBase + obj.Type() + "/" + obj.ExternalIDFieldName() + "/" + obj.ExternalID())
	respData, err := obj.client().httpRequest(http.MethodPatch, url, bytes.NewReader(reqData))
	if err != nil {
		log.Println(logPrefix, "failed to process http request,", err)
		return nil
	}

	// Upsert returns with 201 and id in response if a new record is created. If a record is updated, it returns
	// a 204 with an empty response
	if len(respData) > 0 {
		err = obj.setIDFromResponseData(respData)
		if err != nil {
			log.Println(logPrefix, "failed to parse response,", err)
			return nil
		}
	}

	return obj
}

// Delete deletes an SObject record identified by external ID. nil is returned if the operation completes successfully;
// otherwise an error is returned
func (obj *SObject) Delete(id ...string) error {
	if obj.Type() == "" || obj.client() == nil {
		// Sanity check
		return ErrFailure
	}

	oid := obj.ID()
	if id != nil {
		oid = id[0]
	}
	if oid == "" {
		return ErrFailure
	}

	url := obj.client().makeURL("sobjects/" + obj.Type() + "/" + obj.ID())
	log.Println(url)
	_, err := obj.client().httpRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	return nil
}

// Type returns the type, or sometimes referred to as name, of an SObject.
func (obj *SObject) Type() string {
	attributes := obj.AttributesField()
	if attributes == nil {
		return ""
	}
	return attributes.Type
}

// ID returns the external ID of the SObject.
func (obj *SObject) ID() string {
	return obj.StringField(sobjectIDKey)
}

// ExternalIDField returns the external ID field of the SObject.
func (obj *SObject) ExternalIDFieldName() string {
	return obj.StringField(sobjectExternalIDFieldNameKey)
}

// ExternalID returns the external ID of the SObject.
func (obj *SObject) ExternalID() string {
	return obj.StringField(obj.ExternalIDFieldName())
}

// StringField accesses a field in the SObject as string. Empty string is returned if the field doesn't exist.
func (obj *SObject) StringField(key string) string {
	value := obj.InterfaceField(key)
	switch value.(type) {
	case string:
		return value.(string)
	default:
		return ""
	}
}

// SObjectField accesses a field in the SObject as another SObject. This is only applicable if the field is an external
// ID to another object. The typeName of the SObject must be provided. <nil> is returned if the field is empty.
func (obj *SObject) SObjectField(typeName, key string) *SObject {
	// First check if there's an associated ID directly.
	oid := obj.StringField(key)
	if oid != "" {
		object := &SObject{}
		object.setClient(obj.client())
		object.setType(typeName)
		object.setID(oid)
		return object
	}

	// Secondly, check if this could be a linked object, which doesn't have an ID but has the attributes.
	linkedObjRaw := obj.InterfaceField(key)
	linkedObjMapper, ok := linkedObjRaw.(map[string]interface{})
	if !ok {
		return nil
	}
	attrs, ok := linkedObjMapper[sobjectAttributesKey].(map[string]interface{})
	if !ok {
		return nil
	}

	// Reusing typeName here, which is ok
	typeName, _ = attrs["type"].(string)
	url, _ := attrs["url"].(string)
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

	object := obj.client().SObject(typeName)
	object.setID(oid)
	for key, val := range linkedObjMapper {
		object.Set(key, val)
	}

	return object
}

// InterfaceField accesses a field in the SObject as raw interface. This allows access to any type of fields.
func (obj *SObject) InterfaceField(key string) interface{} {
	return (*obj)[key]
}

// AttributesField returns a read-only copy of the attributes field of an SObject.
func (obj *SObject) AttributesField() *SObjectAttributes {
	attributes := obj.InterfaceField(sobjectAttributesKey)

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
func (obj *SObject) Set(key string, value interface{}) *SObject {
	(*obj)[key] = value
	return obj
}

// client returns the associated Client with the SObject.
func (obj *SObject) client() *Client {
	client := obj.InterfaceField(sobjectClientKey)
	switch client.(type) {
	case *Client:
		return client.(*Client)
	default:
		return nil
	}
}

// setClient sets the associated Client with the SObject.
func (obj *SObject) setClient(client *Client) {
	(*obj)[sobjectClientKey] = client
}

// setType sets the type, or name for the SObject.
func (obj *SObject) setType(typeName string) {
	attributes := obj.InterfaceField(sobjectAttributesKey)
	switch attributes.(type) {
	case SObjectAttributes:
		attrs := obj.AttributesField()
		attrs.Type = typeName
		(*obj)[sobjectAttributesKey] = *attrs
	default:
		(*obj)[sobjectAttributesKey] = SObjectAttributes{
			Type: typeName,
		}
	}
}

// setID sets the external ID for the SObject.
func (obj *SObject) setID(id string) {
	(*obj)[sobjectIDKey] = id
}

// makeCopy copies the fields of an SObject to a new map without metadata fields.
func (obj *SObject) makeCopy() map[string]interface{} {
	stripped := make(map[string]interface{})
	for key, val := range *obj {
		if key == sobjectClientKey ||
			key == sobjectAttributesKey ||
			key == sobjectIDKey ||
			key == sobjectExternalIDFieldNameKey ||
			key == obj.ExternalIDFieldName() {
			continue
		}
		stripped[key] = val
	}
	for _, key := range blacklistedUpdateFields {
		delete(stripped, key)
	}
	return stripped
}

func (obj *SObject) setIDFromResponseData(respData []byte) error {
	// Use an anonymous struct to parse the result if any. This might need to be changed if the result should
	// be returned to the caller in some manner, especially if the client would like to decode the errors.
	var respVal struct {
		ID      string `json:"id"`
		Success bool   `json:"success"`
	}
	err := json.Unmarshal(respData, &respVal)
	if err != nil {
		log.Println(logPrefix, "failed to process response data,", err)
		return err
	}

	if !respVal.Success || respVal.ID == "" {
		log.Println(logPrefix, "unsuccessful")
		return errors.New("request was unsuccessful")
	}

	obj.setID(respVal.ID)
	return nil
}
