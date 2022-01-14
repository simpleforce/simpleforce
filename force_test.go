package simpleforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPClient_Query(t *testing.T) {
	assert := assert.New(t)

	query := "SELECT Id FROM Account"
	var res *QueryResult

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.Equal(query, r.URL.Query().Get("q"))

		err := json.NewEncoder(w).Encode(res)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sObj := NewSObject("").
		Set("Foo", "bar")

	res = &QueryResult{
		Records: []*SObject{sObj},
	}

	actualRes, err := client.Query(query, "")
	assert.NoError(err)
	assert.Equal(res, actualRes)
}

func TestHTTPClient_Query_nextRecordsURL(t *testing.T) {
	assert := assert.New(t)

	query := "SELECT Id FROM Account"
	nextRecordsURL := "/foo/bar"
	var res *QueryResult

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.False(r.URL.Query().Has("q"))
		assert.Equal(nextRecordsURL, r.URL.Path)

		err := json.NewEncoder(w).Encode(res)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sObj := NewSObject("").
		Set("Foo", "bar")

	res = &QueryResult{
		NextRecordsURL: nextRecordsURL,
		Records:        []*SObject{sObj},
	}

	actualRes, err := client.Query(query, nextRecordsURL)
	assert.NoError(err)
	assert.Equal(res, actualRes)
}

func TestHTTPClient_DescribeSObject(t *testing.T) {
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

	sobj := NewSObject("Case")

	meta, err := client.DescribeSObject(sobj)
	assert.NoError(err)
	assert.NotNil(meta)

	name := (*meta)["name"]
	assert.Equal("Case", name)
}

func TestHTTPClient_Get(t *testing.T) {
	assert := assert.New(t)

	id := "object1"
	ownerID := "owner1"
	objType := "Case"

	sobj := NewSObject(objType).
		SetID(id).
		Set("OwnerId", ownerID)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodGet)
		assert.Contains(r.URL.Path, "sobjects/"+objType+"/"+id)

		err := json.NewEncoder(w).Encode(sobj)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	err := client.GetSObject(sobj)
	assert.NoError(err)
	assert.NotNil(sobj)

	assert.Equal(ownerID, sobj.StringField("OwnerId"))
	assert.Equal(objType, sobj.Type())
}

func TestHTTPClient_Create(t *testing.T) {
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
		assert.NotContains(r.Header, duplicateRuleHeader)

		err := json.NewEncoder(w).Encode(res)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sobj := NewSObject(objType).
		SetID(id).
		Set("OwnerId", ownerID)

	err := client.CreateSObject(sobj, nil, false)
	assert.NoError(err)
	assert.NotNil(sobj)

	assert.Equal(ownerID, sobj.StringField("OwnerId"))
	assert.Equal(objType, sobj.Type())
}

func TestHTTPClient_Create_allow_duplicates(t *testing.T) {
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
		assert.Equal(r.Header.Get(duplicateRuleHeader), "allowSave=true")

		err := json.NewEncoder(w).Encode(res)
		assert.NoError(err)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sobj := NewSObject(objType).
		SetID(id).
		Set("OwnerId", ownerID)

	err := client.CreateSObject(sobj, nil, true)
	assert.NoError(err)
	assert.NotNil(sobj)

	assert.Equal(ownerID, sobj.StringField("OwnerId"))
	assert.Equal(objType, sobj.Type())
}

func TestHTTPClient_Update(t *testing.T) {
	assert := assert.New(t)

	id := "object1"
	ownerID := "owner1"
	objType := "Case"

	sobj := NewSObject(objType).
		SetID(id).
		Set("OwnerId", ownerID)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodPatch)
		assert.Contains(r.URL.Path, "sobjects/"+objType+"/"+id)

		o := &SObject{}
		err := json.NewDecoder(r.Body).Decode(o)
		assert.NoError(err)

		assert.Equal(sobj.StringField("OwnerId"), o.StringField("OwnerId"))
		assert.Equal("bar", o.StringField("Foo"))
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	sobj.Set("Foo", "bar")

	err := client.UpdateSObject(sobj, nil)
	assert.NoError(err)

	assert.Equal(ownerID, sobj.StringField("OwnerId"))
	assert.Equal(objType, sobj.Type())
}

func TestHTTPClient_Upsert(t *testing.T) {
	assert := assert.New(t)

	ownerID := "owner1"
	objType := "Case"

	idField := "Baz"
	idValue := "cat"

	sobj := NewSObject(objType).
		Set("OwnerId", ownerID).
		Set("Foo", "bar")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodPatch)

		o := &SObject{}
		err := json.NewDecoder(r.Body).Decode(o)
		assert.NoError(err)

		assert.Contains(r.URL.Path, "sobjects/"+objType+"/"+idField+"/"+idValue)

		assert.Equal(sobj.StringField("OwnerId"), o.StringField("OwnerId"))
		assert.Equal("bar", o.StringField("Foo"))
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	err := client.UpsertSObject(sobj, idField, idValue, nil)
	assert.NoError(err)

	assert.Equal(ownerID, sobj.StringField("OwnerId"))
	assert.Equal(objType, sobj.Type())
}

func TestHTTPClient_Delete(t *testing.T) {
	assert := assert.New(t)

	id := "object1"
	objType := "Case"

	sobj := NewSObject(objType).
		SetID(id)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(r.Method, http.MethodDelete)
		assert.Contains(r.URL.Path, "sobjects/"+objType+"/"+id)
	}))

	client := NewHTTPClient(ts.Client(), ts.URL, DefaultAPIVersion)

	err := client.DeleteSObject(sobj)
	assert.NoError(err)
}
