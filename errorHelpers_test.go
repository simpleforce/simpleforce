package simpleforce

import (
	"testing"
)

var expectedError SalesforceError = SalesforceError{
	HttpCode:     417,
	Message:      logPrefix + " Error. http code: 417 Error Message:  something went wrong Error Code: SMTH_WRNG",
	ErrorCode:    "SMTH_WRNG",
	ErrorMessage: "something went wrong",
}

func TestSuccessfulJSONParse(t *testing.T) {
	response := `[
		{
			"message": "something went wrong",
			"errorCode": "SMTH_WRNG"
		}
	]`

	err := ParseSalesforceError(417, []byte(response))
	if err != expectedError {
		t.Errorf("failed to parse JSON error, got %s", err)
	}
}

func TestSuccessfulXMLParse(t *testing.T) {
	response := `
		<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
			<s:Body>
				<s:Fault>
					<faultcode>SMTH_WRNG</faultcode>
					<faultstring xml:lang="en-US">something went wrong</faultstring>
				</s:Fault>
			</s:Body>
		</s:Envelope>
	`
	err := ParseSalesforceError(417, []byte(response))
	if err != expectedError {
		t.Errorf("failed to parse XML error, got %s", err)
	}
}

func TestUnsuccessfulParse(t *testing.T) {
	response := "surprise!"
	unknownError := SalesforceError{
		HttpCode: 417,
		Message:  "surprise!",
	}

	err := ParseSalesforceError(417, []byte(response))
	if err != unknownError {
		t.Errorf("failed to parse unknown error, got %s", err)
	}
}
