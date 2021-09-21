# simpleforce

A simple Golang client for Salesforce

[![GoDoc](https://godoc.org/github.com/eleanorhealth/simpleforce?status.svg)](https://godoc.org/github.com/eleanorhealth/simpleforce)

## Features

`simpleforce` is a library written in Go (Golang) that connects to Salesforce via the REST API.
Currently, the following functions are implemented and more features could be added based on need:

* Execute SOQL queries
* Get records via record (sobject) type and ID
* Create records
* Update records
* Delete records
* Download a file

Most of the implementation referenced Salesforce documentation here: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/intro_what_is_rest_api.htm

## Installation

`simpleforce` can be acquired as any other Go libraries via `go get`:

```
go get github.com/eleanorhealth/simpleforce
```

## Quick Start

### Setup the Client

Create an `HTTPClient` instance with the
`NewHTTPClient` function with an oauth2 configured HTTP client and the proper endpoint URL:

```go
package main

import (
	"context"
	"log"

	"github.com/eleanorhealth/simpleforce"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	oauth2Config := &clientcredentials.Config{
		ClientID:     "<salesforce client ID",
		ClientSecret: "<salesforce client secret",
		TokenURL:     "https://test.salesforce.com/services/oauth2/token",
		EndpointParams: map[string][]string{
			"grant_type": {"password"},
			"username":   {"<salesforce username"},
			"password":   {"<salesforce password + security token"},
		},
		AuthStyle: oauth2.AuthStyleInParams,
	}

	httpClient := oauth2Config.Client(context.Background())

	client := simpleforce.NewHTTPClient(httpClient, "<salesforce base URL>", simpleforce.DefaultAPIVersion)
```

### Execute a SOQL Query

The `client` provides an interface to run an SOQL Query. Refer to 
https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/dome_query.htm
for more details about SOQL.

```go
package main

import (
    "fmt"
    "github.com/eleanorhealth/simpleforce"
)

func main() {
	client := simpleforce.NewHTTPClient(...)

	q := "Some SOQL Query String"
	var nextRecordsURL string

	for {
		result, err := client.Query(q, nextRecordsURL)
		if err != nil {
			// handle the error
			return
		}
		nextURL = result.NextRecordsURL


		for _, record := range result.Records {
			// access the record as SObjects.
			fmt.Println(record.StringField("SomeField"))
		}

		if res.Done {
			break
		}
	}
}

```

### Work with Records

`SObject` instances are returned as records in the result of `client.Query()` but can also be created manually using `NewSObject()`. `SObject` instances can be created, read, updated, or deleted using the `CreateSObject()`, `GetSObject()`, `UpdateSObject()`, and `DeleteSObject()` methods on `HTTPClient`.

```go
package main

import (
    "fmt"
    "github.com/eleanorhealth/simpleforce"
)

func main() {
	client := simpleforce.NewHTTPClient(...)
	
	// Get an SObject with given type and external ID
	obj := simpleforce.NewSObject("Case").SetID("<some sobject ID>")

	err := client.GetSObject(obj)
	if err != nil {
		log.Fatal(err)
	}
	
	// Attributes are associated with all Salesforce returned SObjects, and can be accessed with the
	// `AttributesField` method.
	attrs := obj.AttributesField()
	if attrs != nil {
	 	fmt.Println(attrs.Type)    // "Case" 
		fmt.Println(attrs.URL)     // "/services/data/v43.0/case/__ID__"
	}

	// For Update(), start with a blank SObject.
	// Set "Id" with an existing ID and any updated fields.
	contractObj := NewSObject("Contract").
		SetID("<some sobject ID">).										// Set the Id to an existing Contact ID.
		Set("FirstName", "New Name")									// Set any updated fields.
		
	err = client.UpdateSObject(contractObj)								// Update the record on Salesforce server.
	if err != nil {
		log.Fatal(err)



	caseObj := NewSObject("Case").SetID("<some sobject ID">)
	err = client.DeleteSObject(caseObj)                                                    // Delete the record from Salesforce server.
	if err != nil {
		log.Fatal(err)
	}
}
```

### Download a File
```go

// Get the content version ID
query := "SELECT Id FROM ContentVersion WHERE ContentDocumentId = '" + YOUR_CONTENTDOCUMENTID + "' AND IsLatest = true"
result, err = client.Query(query, "")
if err != nil {
    // handle error
    return
}

contentVersionID := ""
for _, record := range result.Records {
    contentVersionID = record.StringField("Id")
}

// Download file
downloadFilePath := "/absolute/path/to/yourfile.txt"

err = client.DownloadFile(contentVersionID, downloadFilePath)
if err != nil {
    // handle error
    return
}   
```
