package simpleforce

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrAuthentication = errors.New("authentication failure")
	ErrFailure        = errors.New("general failure")
)

type ErrInvalidSObject struct {
	msg string
}

func (e ErrInvalidSObject) Error() string {
	return fmt.Sprintf("invalid sobject: %s", e.msg)
}

type jsonError []struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

type xmlError struct {
	Message   string `xml:"Body>Fault>faultstring"`
	ErrorCode string `xml:"Body>Fault>faultcode"`
}

func parseSalesforceError(statusCode int, responseBody []byte) (err error) {
	jsonError := jsonError{}
	xmlError := xmlError{}
	err = json.Unmarshal(responseBody, &jsonError)
	if err != nil {
		//Unable to parse json. Try xml
		err = xml.Unmarshal(responseBody, &xmlError)
		if err != nil {
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
