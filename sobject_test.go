package simpleforce

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSObject_AttributesField(t *testing.T) {
	obj := &SObject{}
	if obj.AttributesField() != nil {
		t.Fail()
	}

	obj.setType("Case")
	if obj.AttributesField().Type != "Case" {
		t.Fail()
	}

	obj.setType("")
	if obj.AttributesField().Type != "" {
		t.Fail()
	}
}

func TestSObject_Type(t *testing.T) {
	obj := &SObject{
		sobjectAttributesKey: SObjectAttributes{Type: "Case"},
	}
	if obj.Type() != "Case" {
		t.Fail()
	}

	obj.setType("CaseComment")
	if obj.Type() != "CaseComment" {
		t.Fail()
	}
}

func TestSObject_InterfaceField(t *testing.T) {
	obj := &SObject{}
	if obj.InterfaceField("test_key") != nil {
		t.Fail()
	}

	(*obj)["test_key"] = "hello"
	if obj.InterfaceField("test_key") == nil {
		t.Fail()
	}
}

func TestSObject_SObjectField(t *testing.T) {
	obj := &SObject{
		sobjectAttributesKey: SObjectAttributes{Type: "CaseComment"},
		"ParentId":           "__PARENT_ID__",
	}

	// Positive checks
	caseObj := obj.SObjectField("Case", "ParentId")
	if caseObj.Type() != "Case" {
		log.Println("Type mismatch")
		t.Fail()
	}
	if caseObj.StringField("Id") != "__PARENT_ID__" {
		log.Println("ID mismatch")
		t.Fail()
	}

	// Negative checks
	userObj := obj.SObjectField("User", "OwnerId")
	if userObj != nil {
		log.Println("Nil mismatch")
		t.Fail()
	}
}

func TestSObject_Describe(t *testing.T) {
	assert := assert.New(t)

	res := &SObjectMeta{
		"name": "Case",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.Contains(r.URL.Path, "sobjects/Case/describe")

		err := json.NewEncoder(w).Encode(res)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	meta, err := client.SObject("Case").Describe()
	assert.NoError(err)
	assert.NotNil(meta)

	name := (*meta)["name"]
	assert.Equal("Case", name)
}

func TestSObject_Get(t *testing.T) {
	assert := assert.New(t)

	id := "object1"
	ownerID := "owner1"
	objType := "Case"

	sObj := &SObject{
		"OwnerId": ownerID,
		"attributes": SObjectAttributes{
			Type: objType,
		},
	}
	sObj.setID(id)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.Contains(r.URL.Path, "sobjects/"+objType+"/"+id)

		err := json.NewEncoder(w).Encode(sObj)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sObj.setClient(client)

	sObj, err := sObj.Get()
	assert.NoError(err)
	assert.NotNil(sObj)

	assert.Equal(ownerID, sObj.StringField("OwnerId"))
	assert.Equal(objType, sObj.Type())
}

func TestSObject_Create(t *testing.T) {
	assert := assert.New(t)

	id := "object1"
	ownerID := "owner1"
	objType := "Case"

	res := &createSObjectResponse{
		ID:      id,
		Success: true,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodPost)
		assert.Contains(r.URL.Path, "sobjects/"+objType)

		err := json.NewEncoder(w).Encode(res)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sObj := &SObject{
		"OwnerId": ownerID,
		"attributes": SObjectAttributes{
			Type: objType,
		},
	}
	sObj.setID(id)
	sObj.setClient(client)

	sObj, err := sObj.Create(nil)
	assert.NoError(err)
	assert.NotNil(sObj)

	assert.Equal(ownerID, sObj.StringField("OwnerId"))
	assert.Equal(objType, sObj.Type())
}

func TestSObject_Update(t *testing.T) {
	assert := assert.New(t)

	id := "object1"
	ownerID := "owner1"
	objType := "Case"

	sObj := &SObject{
		"OwnerId": ownerID,
		"attributes": SObjectAttributes{
			Type: objType,
		},
	}
	sObj.setID(id)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodPatch)
		assert.Contains(r.URL.Path, "sobjects/"+objType+"/"+id)

		o := &SObject{}
		err := json.NewDecoder(r.Body).Decode(o)
		assert.NoError(err)

		assert.Equal(sObj.StringField("OwnerId"), o.StringField("OwnerId"))
		assert.Equal("bar", o.StringField("Foo"))
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sObj.setClient(client)
	sObj.Set("Foo", "bar")

	err := sObj.Update(nil)
	assert.NoError(err)

	assert.Equal(ownerID, sObj.StringField("OwnerId"))
	assert.Equal(objType, sObj.Type())
}

func TestSObject_Delete(t *testing.T) {
	assert := assert.New(t)

	id := "object1"
	objType := "Case"

	sObj := &SObject{
		"attributes": SObjectAttributes{
			Type: objType,
		},
	}
	sObj.setID(id)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodDelete)
		assert.Contains(r.URL.Path, "sobjects/"+objType+"/"+id)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sObj.setClient(client)

	err := sObj.Delete()
	assert.NoError(err)
}
