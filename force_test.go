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

	sObj := SObject{
		"Foo": "bar",
	}
	sObj.setClient(client)

	res = &QueryResult{
		Records: []SObject{sObj},
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

	sObj := SObject{
		"Foo": "bar",
	}
	sObj.setClient(client)

	res = &QueryResult{
		NextRecordsURL: nextRecordsURL,
		Records:        []SObject{sObj},
	}

	actualRes, err := client.Query(query, nextRecordsURL)
	assert.NoError(err)
	assert.Equal(res, actualRes)
}
