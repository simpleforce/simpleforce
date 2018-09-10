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
	sfURL   = os.ExpandEnv("${SF_URL}")
)

func TestLogin(t *testing.T) {
	cli := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	err := cli.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
		log.Println(logPrefix, "login failed,", err)
		t.Fail()
	}
	log.Println("SessionID:", cli.sessionID)
}

func TestQuery(t *testing.T) {
	cli := NewClient(sfURL, DefaultClientID, DefaultAPIVersion)
	err := cli.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
		log.Print(logPrefix, "login failed,", err)
		t.Fail()
	}

	q := "SELECT Id,LastModifiedById,LastModifiedDate,ParentId,CommentBody FROM CaseComment"
	result, err := cli.Query(q)
	if err != nil {
		log.Println(logPrefix, "query failed,", err)
		t.Fail()
	}

	for _, record := range result.Records {
		log.Println(record["Id"], record["LastModifiedById"], record["LastModifiedDate"], record["ParentId"], record["CommentBody"])
	}
}

func TestMain(m *testing.M) {
	if sfUser == "" || sfPass == "" || sfToken == "" {
		log.Println(logPrefix, "SF_USER, SF_PASS, or SF_TOKEN environment variables are not set.")
		return
	}

	if sfURL == "" {
		sfURL = DefaultURL
	}
	log.Printf(logPrefix+" using URL:%s, user:%s, pass:%s, token:%s", sfURL, sfUser, sfPass, sfToken)

	m.Run()
}
