package simpleforce

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"

	"github.com/pkg/errors"
)

var (
	// ErrFailure is a generic error if none of the other errors are appropriate.
	ErrFailure = errors.New("general failure")

	// ErrAuthentication is returned when authentication failed.
	ErrAuthentication = errors.New("authentication failure")
)

type jsonError []struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

type xmlError struct {
	Message   string `xml:"Body>Fault>faultstring"`
	ErrorCode string `xml:"Body>Fault>faultcode"`
}

//Need to get information out of this package.
func ParseSalesforceError(statusCode int, responseBody []byte) (err error) {
	jsonError := jsonError{}
	err = json.Unmarshal(responseBody, &jsonError)
	if err == nil {
		return fmt.Errorf(logPrefix+" Error. http code: %v Error Message:  %v Error Code: %v", statusCode, jsonError[0].Message, jsonError[0].ErrorCode)
	}

	xmlError := xmlError{}
	err = xml.Unmarshal(responseBody, &xmlError)
	if err == nil {
		return fmt.Errorf(logPrefix+" Error. http code: %v Error Message:  %v Error Code: %v", statusCode, xmlError.Message, xmlError.ErrorCode)
	}

	log.Println("ERROR UNMARSHALLING: ", err)
	return ErrFailure
}
