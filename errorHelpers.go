package simpleforce

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

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

type SalesforceError struct {
	Message      string
	HttpCode     int
	ErrorCode    string
	ErrorMessage string
}

func (err SalesforceError) Error() string {
	return err.Message
}

//Need to get information out of this package.
func ParseSalesforceError(statusCode int, responseBody []byte) (err error) {
	jsonError := jsonError{}
	err = json.Unmarshal(responseBody, &jsonError)
	if err == nil {
		return SalesforceError{
			Message: fmt.Sprintf(
				logPrefix+" Error. http code: %v Error Message:  %v Error Code: %v",
				statusCode, jsonError[0].Message, jsonError[0].ErrorCode,
			),
			HttpCode:     statusCode,
			ErrorCode:    jsonError[0].ErrorCode,
			ErrorMessage: jsonError[0].Message,
		}
	}

	xmlError := xmlError{}
	err = xml.Unmarshal(responseBody, &xmlError)
	if err == nil {
		return SalesforceError{
			Message: fmt.Sprintf(
				logPrefix+" Error. http code: %v Error Message:  %v Error Code: %v",
				statusCode, xmlError.Message, xmlError.ErrorCode,
			),
			HttpCode:     statusCode,
			ErrorCode:    xmlError.ErrorCode,
			ErrorMessage: xmlError.Message,
		}
	}

	return SalesforceError{
		Message:  string(responseBody),
		HttpCode: statusCode,
	}
}
