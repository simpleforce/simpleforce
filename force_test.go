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

func checkCredentials(t *testing.T) {
	if sfUser == "" || sfPass == "" || sfToken == "" {
		log.Println(logPrefix, "SF_USER, SF_PASS, or SF_TOKEN environment variables are not set.")
		t.Skip()
	}
}

func TestLogin(t *testing.T) {
	checkCredentials(t)
	log.Printf(logPrefix+" using URL:%s, user:%s, pass:%s, token:%s", sfURL, sfUser, sfPass, sfToken)

	cli := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	err := cli.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
		log.Println(logPrefix, "login failed,", err)
		t.Fail()
		return
	}
	log.Println("SessionID:", cli.sessionID)
}

func TestQuery(t *testing.T) {
	checkCredentials(t)
	log.Printf(logPrefix+" using URL:%s, user:%s, pass:%s, token:%s", sfURL, sfUser, sfPass, sfToken)

	cli := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	err := cli.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
		log.Println(logPrefix, "login failed,", err)
		t.Fail()
		return
	}

	q := "SELECT Id,LastModifiedById,LastModifiedDate,ParentId,CommentBody FROM CaseComment"
	result, err := cli.Query(q)
	if err != nil {
		log.Println(logPrefix, "query failed,", err)
		t.Fail()
		return
	}

	log.Println(result.TotalSize, result.Done, result.NextRecordsURL)
	for _, record := range result.Records {
		log.Println(record.StringField("Id"), record["LastModifiedById"], record["LastModifiedDate"], record["ParentId"], record["CommentBody"])
	}

	if result.NextRecordsURL != "" {
		result, err := cli.Query(result.NextRecordsURL)
		if err != nil {
			log.Println(logPrefix, "query more failed,", err)
			t.Fail()
			return
		}
		log.Println(result.TotalSize, result.Done, result.NextRecordsURL)
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
