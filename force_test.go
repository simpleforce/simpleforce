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
)

func TestLogin(t *testing.T) {
    log.Printf(logPrefix + " using user:%s, pass:%s, token:%s", sfUser, sfPass, sfToken)
    if sfUser == "" || sfPass == "" || sfToken == "" {
        log.Println(logPrefix, "SF_USER, SF_PASS, or SF_TOKEN environment variables are not set.")
        t.SkipNow()
    }
	cli := NewClient(DefaultURL, DefaultClientID, DefaultAPIVersion)
	err := cli.LoginPassword(sfUser, sfPass, sfToken)
	if err != nil {
	    log.Println(logPrefix, "login failed,", err)
	    t.Fail()
    }
}
