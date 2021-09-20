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

`SObject` instances are created by `client` instance, either through the return values of `client.Query()`
or through `client.SObject()` directly. Once an `SObject` instance is created, it can be used to Create, Delete, Update,
or Get records through the Salesforce REST API. Here are some examples of using `SObject` instances to work on the
records.

```go
package main

import (
    "fmt"
    "github.com/eleanorhealth/simpleforce"
)

func main() {
	client := simpleforce.NewHTTPClient(...)
	
	// Get an SObject with given type and external ID
	obj, err := client.SObject("Case").Get("<some sobject ID>")
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
	
	// Linked objects can be accessed with the `SObjectField` method.
	userObj := obj.SObjectField("User", "CreatedById")
	if userObj == nil {
		log.Fatal(`Object doesn't exist, or field "CreatedById" is invalid`)
	}
	
	// Linked objects returned normally contains the type and ID field only. A `Get` operation is needed to
	// retrieve all the information of linked objects.
	fmt.Println(userObj.StringField("Name"))    // FAIL: fields are not populated.
	
	// If an SObject instance already has an ID (e.g. linked object), `Get` can retrieve the object directly without
	// parameter.
	userObj, err = userObj.Get()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(userObj.StringField("Name"))    // SUCCESS: returns the name of the user.
	
	// For Update(), start with a blank SObject.
	// Set "Id" with an existing ID and any updated fields.
	err = client.SObject("Contact").									// Create an empty object of type "Contact".
		Set("Id", "__ID__").											// Set the Id to an existing Contact ID.
		Set("FirstName", "New Name").									// Set any updated fields.
		Update()														// Update the record on Salesforce server.
	if err != nil {
		log.Fatal(err)


	// Many SObject methods return the instance of the SObject, allowing chained access and operations to the
	// object. In the following example, all methods, except "Delete", returns *SObject so that the next method
	// can be invoked on the returned value directly.
	//
	// Delete() methods returns `error` instead, as Delete is supposed to delete the record from the server.
	err := client.SObject("Case").                               		// Create an empty object of type "Case"
    		Set("Subject", "Case created by simpleforce").              // Set the "Subject" field.
	        Set("Comments", "Case commented by simpleforce").           // Set the "Comments" field.
    		Create().                                                   // Create the record on Salesforce server.
    		Get().                                                      // Refresh the fields from Salesforce server.
    		Delete()                                                    // Delete the record from Salesforce server.
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
