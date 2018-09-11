package simpleforce

import (
	"log"
	"os"
	"testing"
)

var (
	sfUser  = os.ExpandEnv("${SF_USER}")
	sfPass  = os.ExpandEnv("${SF_PASS}")
	sfToken = os.ExpandEnv("${SF_TOKEN}")
	sfURL   = func() string {
		if os.ExpandEnv("${SF_URL}") != "" {
			return os.ExpandEnv("${SF_URL}")
		} else {
			return DefaultURL
		}
	}()
)

func checkCredentialsAndSkip(t *testing.T) {
	if sfUser == "" || sfPass == "" || sfToken == "" {
		log.Println(logPrefix, "SF_USER, SF_PASS, or SF_TOKEN environment variables are not set.")
		t.Skip()
	}
}

func requireClient(t *testing.T, skippable bool) *Client {
	if skippable {
		checkCredentialsAndSkip(t)
	}

	client := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	if client == nil {
		t.Fatal()
	}
	err := client.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
		t.Fatal()
	}
	return client
}

func TestClient_LoginPassword(t *testing.T) {
	client := requireClient(t, true)
	log.Println(logPrefix, "SessionID:", client.sessionID)
}

func TestClient_Query(t *testing.T) {
	client := requireClient(t, true)

	q := "SELECT Id,LastModifiedById,LastModifiedDate,ParentId,CommentBody FROM CaseComment"
	result, err := client.Query(q)
	if err != nil {
		log.Println(logPrefix, "query failed,", err)
		t.Fail()
	}

	log.Println(logPrefix, result.TotalSize, result.Done, result.NextRecordsURL)
	if result.TotalSize < 1 {
		log.Println(logPrefix, "no records returned.")
		t.Fail()
	}
	for _, record := range result.Records {
		log.Println(logPrefix, record.StringField("Id"), record["LastModifiedById"], record["LastModifiedDate"], record["ParentId"], record["CommentBody"])
		if record.Type() != "CaseComment" {
			t.Fail()
		}
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
