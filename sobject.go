package simpleforce

import (
	"bytes"
	"encoding/json"
	"log"
)

const (
	sobjectClientKey     = "__client__" // private attribute added to locate client instance.
	sobjectAttributesKey = "attributes" // points to the attributes structure which should be common to all SObjects.
	sobjectIDKey         = "Id"
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
	data, err := obj.client().httpRequest("GET", url, nil)
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
	if id != nil {
		oid = id[0]
	}
	if oid == "" {
		log.Println(logPrefix, "object id not found.")
		return nil
	}

	url := obj.client().makeURL("sobjects/" + obj.Type() + "/" + oid)
	data, err := obj.client().httpRequest("GET", url, nil)
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
// If the creation is successful, a new SObject instance is returned with the external ID and type of the created
// object; <nil> is returned if the creation is failed.
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
	respData, err := obj.client().httpRequest("POST", url, bytes.NewReader(reqData))
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

	// Upon successful invocation, a new SObject with client, ID, and Type configured is returned.
	result := &SObject{}
	result.setClient(obj.client())
	result.setType(obj.Type())
	result.setID(respVal.ID)

	return result
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
	oid := obj.StringField(key)
	if oid == "" {
		return nil
	}

	object := &SObject{}
	object.setClient(obj.client())
	object.setType(typeName)
	object.setID(oid)

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
			key == sobjectIDKey {
			continue
		}
		stripped[key] = val
	}
	return stripped
}
