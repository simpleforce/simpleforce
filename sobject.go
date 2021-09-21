package simpleforce

const (
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

func NewSObject(typeName string) *SObject {
	obj := &SObject{}

	if len(typeName) > 0 {
		obj.setType(typeName)
	}

	return obj
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

// SetID sets the external ID for the SObject.
func (s *SObject) SetID(id string) *SObject {
	(*s)[sobjectIDKey] = id

	return s
}

// makeCopy copies the fields of an SObject to a new map without metadata fields.
func (s *SObject) makeCopy(blacklistedFields []string) map[string]interface{} {
	stripped := make(map[string]interface{})

	for key, val := range *s {
		if key == sobjectAttributesKey ||
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
