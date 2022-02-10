# simpleforce

A simple Golang client for Salesforce

[![GoDoc](https://godoc.org/github.com/simpleforce/simpleforce?status.svg)](https://godoc.org/github.com/simpleforce/simpleforce)

## Features

`simpleforce` is a library written in Go (Golang) that connects to Salesforce via the REST and Tooling APIs.
Currently, the following functions are implemented and more features could be added based on need:

- Execute SOQL queries
- Get records via record (sobject) type and ID
- Create records
- Update records
- Delete records
- Upsert (create or update) records based on an external ID
- Download a file
- Execute anonymous apex
- Send request to a custom Apex Rest endpoint

Most of the implementation referenced Salesforce documentation here: https://developer.salesforce.com/docs/atlas.en-us.214.0.api_rest.meta/api_rest/intro_what_is_rest_api.htm

## Installation

`simpleforce` can be acquired as any other Go libraries via `go get`:

```
go get github.com/simpleforce/simpleforce
```

## Quick Start

### Setup the Client

A `client` instance is the main entrance to access Salesforce using simpleforce. Create a `client` instance with the
`NewClient` function, with the proper endpoint URL:

```go
package main

import "github.com/simpleforce/simpleforce"

var (
	sfURL      = "Custom or instance URL, for example, 'https://na01.salesforce.com/'"
	sfUser     = "Username of the Salesforce account."
	sfPassword = "Password of the Salesforce account."
	sfToken    = "Security token, could be omitted if Trusted IP is configured."
)

func createClient() *simpleforce.Client {
	client := simpleforce.NewClient(sfURL, simpleforce.DefaultClientID, simpleforce.DefaultAPIVersion)
	if client == nil {
		// handle the error

		return nil
	}

	err := client.LoginPassword(sfUser, sfPassword, sfToken)
	if err != nil {
		// handle the error

		return nil
	}

	// Do some other stuff with the client instance if needed.

	return client
}
```

### Execute a SOQL Query

The `client` provides an interface to run an SOQL Query. Refer to
https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_query.htm
for more details about SOQL.

```go
package main

import (
    "fmt"
    "github.com/simpleforce/simpleforce"
)

func Query() {
	client := simpleforce.NewClient(...)
	client.LoginPassword(...)

	q := "Some SOQL Query String"
	result, err := client.Query(q) // Note: for Tooling API, use client.Tooling().Query(q)
	if err != nil {
		// handle the error
		return
	}

	for _, record := range result.Records {
		// access the record as SObjects.
		fmt.Println(record)
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
    "github.com/simpleforce/simpleforce"
)

func WorkWithRecords() {
	client := simpleforce.NewClient(...)
	client.LoginPassword(...)

	// Get an SObject with given type and external ID
	obj := client.SObject("Case").Get("__ID__")
	if obj == nil {
		// Object doesn't exist, handle the error
		return
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
		// Object doesn't exist, or field "CreatedById" is invalid.
		return
	}

	// Linked objects returned normally contains the type and ID field only. A `Get` operation is needed to
	// retrieve all the information of linked objects.
	fmt.Println(userObj.StringField("Name"))    // FAIL: fields are not populated.

	// If an SObject instance already has an ID (e.g. linked object), `Get` can retrieve the object directly without
	// parameter.
	userObj.Get()
	fmt.Println(userObj.StringField("Name"))    // SUCCESS: returns the name of the user.

	// For Update(), start with a blank SObject.
	// Set "Id" with an existing ID and any updated fields.
	//
	// Update() will return the updated object, or nil and print an error.
	updateObj := client.SObject("Contact").								// Create an empty object of type "Contact".
		Set("Id", "__ID__").											// Set the Id to an existing Contact ID.
		Set("FirstName", "New Name").									// Set any updated fields.
		Update()														// Update the record on Salesforce server.
	fmt.Println(updateObj)

	// For Upsert(), start with a blank SObject.
	// Upsert will create the object if it does not already exist and will update the object if it already exists.
	// Set "ExternalIDField" to the name of your external ID field
	// and populate that field with an your external ID and any updated fields.
	//
	// Upsert() will return the updated object, or nil and print an error.
	upsertObj := client.SObject("Contact").						// Create an empty object of type "Contact".
		Set("ExternalIDField", "customExtIdField__c").	// Set the ExternalIDField to the name of your external ID field.
		Set("customExtIdField__c", "__ExtID__").				// Set the specified ID field to your external ID.
		Set("FirstName", "New Name").										// Set any updated fields.
		Upsert()																				// Update the record on Salesforce server.
	fmt.Println(upsertObj)

	// Many SObject methods return the instance of the SObject, allowing chained access and operations to the
	// object. In the following example, all methods, except "Delete", returns *SObject so that the next method
	// can be invoked on the returned value directly.
	//
	// Delete() methods returns `error` instead, as Delete is supposed to delete the record from the server.
	err := client.SObject("Case").                               // Create an empty object of type "Case"
    		Set("Subject", "Case created by simpleforce").              // Set the "Subject" field.
	        Set("Comments", "Case commented by simpleforce").           // Set the "Comments" field.
    		Create().                                                   // Create the record on Salesforce server.
    		Get().                                                      // Refresh the fields from Salesforce server.
    		Delete()                                                    // Delete the record from Salesforce server.
	fmt.Println(err)
}
```

### Download a File

```go
// Setup client and login
// ...

// Get the content version ID
query := "SELECT Id FROM ContentVersion WHERE ContentDocumentId = '" + YOUR_CONTENTDOCUMENTID + "' AND IsLatest = true"
result, err = client.Query(query)
if err != nil {
    // handle the error
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

### Execute Anonymous Apex

```go
// Setup client and login
// ...

result, err := client.ExecuteAnonymous("System.debug('test anonymous apex');")
if err != nil {
    // handle error
    return
}
```

## Development and Unit Test

A set of unit test cases are provided to validate the basic functions of simpleforce. Please do not run these
unit tests with a production instance of Salesforce as it would create, modify and delete data from the provided
Salesforce account.

The unit test requires a custom field `customExtIdField__c` to be present on the Type `Case` in your Salesforce setup.

## License and Acknowledgement

This package is released under BSD license. Part of the code referenced the simple-salesforce
(https://github.com/simple-salesforce/simple-salesforce) project and the credit goes to Nick Catalano and the community
maintaining simple-salesforce.

Contributions are welcome, cheers :-)
