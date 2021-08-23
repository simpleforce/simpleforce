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
	xmlError := xmlError{}
	err = json.Unmarshal(responseBody, &jsonError)
	if err != nil {
		//Unable to parse json. Try xml
		err = xml.Unmarshal(responseBody, &xmlError)
		if err != nil {
			//Unable to parse json or XML
			log.Println("ERROR UNMARSHALLING: ", err)
			return ErrFailure
		}
		//successfully parsed XML:
		message := fmt.Sprintf(logPrefix+" Error. http code: %v Error Message:  %v Error Code: %v", statusCode, xmlError.Message, xmlError.ErrorCode)
		err = errors.New(message)
		return err
	} else {
		//Successfully parsed json error:
		message := fmt.Sprintf(logPrefix+" Error. http code: %v Error Message:  %v Error Code: %v", statusCode, jsonError[0].Message, jsonError[0].ErrorCode)
		err = errors.New(message)
		return err
	}
}
