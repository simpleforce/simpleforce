package simpleforce

import (
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSObject_AttributesField(t *testing.T) {
	obj := &SObject{}
	if obj.AttributesField() != nil {
		t.Fail()
	}

	obj.setType("Case")
	if obj.AttributesField().Type != "Case" {
		t.Fail()
	}

	obj.setType("")
	if obj.AttributesField().Type != "" {
		t.Fail()
	}
}

func TestSObject_Type(t *testing.T) {
	obj := &SObject{
		sobjectAttributesKey: SObjectAttributes{Type: "Case"},
	}
	if obj.Type() != "Case" {
		t.Fail()
	}

	obj.setType("CaseComment")
	if obj.Type() != "CaseComment" {
		t.Fail()
	}
}

func TestSObject_InterfaceField(t *testing.T) {
	obj := &SObject{}
	if obj.InterfaceField("test_key") != nil {
		t.Fail()
	}

	(*obj)["test_key"] = "hello"
	if obj.InterfaceField("test_key") == nil {
		t.Fail()
	}
}

func TestSObject_SObjectField(t *testing.T) {
	obj := &SObject{
		sobjectAttributesKey: SObjectAttributes{Type: "CaseComment"},
		"ParentId":           "__PARENT_ID__",
	}

	// Positive checks
	caseObj := obj.SObjectField("Case", "ParentId")
	if caseObj.Type() != "Case" {
		log.Println("Type mismatch")
		t.Fail()
	}
	if caseObj.StringField("Id") != "__PARENT_ID__" {
		log.Println("ID mismatch")
		t.Fail()
	}

	// Negative checks
	userObj := obj.SObjectField("User", "OwnerId")
	if userObj != nil {
		log.Println("Nil mismatch")
		t.Fail()
	}
}

func TestSObject_Describe(t *testing.T) {
	client := requireClient(t, true)
	meta := client.SObject("Case").Describe()
	if meta == nil {
		t.FailNow()
	} else {
		if (*meta)["name"].(string) != "Case" {
			t.Fail()
		}
	}
}

func TestSObject_Get(t *testing.T) {
	client := requireClient(t, true)

	// Search for a valid Case ID first.
	queryResult, err := client.Query("SELECT Id,OwnerId,Subject FROM CASE")
	if err != nil || queryResult == nil {
		log.Println(logPrefix, "query failed,", err)
		t.FailNow()
	}
	if queryResult.TotalSize < 1 {
		t.FailNow()
	}
	oid := queryResult.Records[0].ID()
	ownerID := queryResult.Records[0].StringField("OwnerId")

	// Positive
	obj := client.SObject("Case").Get(oid)
	if obj.ID() != oid || obj.StringField("OwnerId") != ownerID {
		t.Fail()
	}

	// Positive 2
	obj = client.SObject("Case")
	if obj.StringField("OwnerId") != "" {
		t.Fail()
	}
	obj.setID(oid)
	obj.Get()
	if obj.ID() != oid || obj.StringField("OwnerId") != ownerID {
		t.Fail()
	}

	// Negative 1
	obj = client.SObject("Case").Get("non-exist-id")
	if obj != nil {
		t.Fail()
	}

	// Negative 2
	obj = &SObject{}
	if obj.Get() != nil {
		t.Fail()
	}
}

func TestSObject_Create(t *testing.T) {
	client := requireClient(t, true)

	// Positive
	case1 := client.SObject("Case")
	case1Result := case1.Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Set("Comments", "This case is created by simpleforce").
		Create()
	if case1Result == nil || case1Result.ID() == "" || case1Result.Type() != case1.Type() {
		t.Fail()
	} else {
		log.Println(logPrefix, "Case created,", case1Result.Get().StringField("CaseNumber"))
	}

	// Positive 2
	caseComment1 := client.SObject("CaseComment")
	caseComment1Result := caseComment1.Set("ParentId", case1Result.ID()).
		Set("CommentBody", "This comment is created by simpleforce & used for testing").
		Set("IsPublished", true).
		Create()
	if caseComment1Result.Get().SObjectField("Case", "ParentId").ID() != case1Result.ID() {
		t.Fail()
	} else {
		log.Println(logPrefix, "CaseComment created,", caseComment1Result.ID())
	}

	// Negative: object without type.
	obj := client.SObject()
	if obj.Create() != nil {
		t.Fail()
	}

	// Negative: object without client.
	obj = &SObject{}
	if obj.Create() != nil {
		t.Fail()
	}

	// Negative: Invalid type
	obj = client.SObject("__SOME_INVALID_TYPE__")
	if obj.Create() != nil {
		t.Fail()
	}

	// Negative: Invalid field
	obj = client.SObject("Case").Set("__SOME_INVALID_FIELD__", "")
	if obj.Create() != nil {
		t.Fail()
	}
}

func TestSObject_Update(t *testing.T) {
	client := requireClient(t, true)

	// Positive
	if client.SObject("Case").
		Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Create().
		Set("Subject", "Case subject updated by simpleforce").
		Update().
		Get().
		StringField("Subject") != "Case subject updated by simpleforce" {
		t.Fail()
	}
}

func TestSObject_Upsert(t *testing.T) {
	client := requireClient(t, true)

	// Positive create new object through upsert
	case1 := client.SObject("Case")
	case1Result := case1.Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Set("Comments", "This case is created by simpleforce").
		Set("customExtIdField__c", uuid.NewString()).
		Set("ExternalIDField", "customExtIdField__c").
		Upsert()
	if case1Result == nil || case1Result.ID() == "" || case1Result.Type() != case1.Type() {
		t.Fail()
	} else {
		log.Println(logPrefix, "Case created,", case1Result.Get().StringField("CaseNumber"))
	}

	// Positive update existing object through upsert
	case2 := client.SObject("Case").
		Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Set("customExtIdField__c", uuid.NewString())
	case2Result := case2.Create()
	case2.
		Set("Subject", "Case subject updated by simpleforce").
		Set("ExternalIDField", "customExtIdField__c").
		Upsert()
	if case2Result.Get().StringField("Subject") != "Case subject updated by simpleforce" {
		t.Fail()
	} else {
		log.Println(logPrefix, "Case updated,", case2Result.Get().StringField("CaseNumber"))
	}

	// Negative: object without type.
	obj := client.SObject()
	if obj.Upsert() != nil {
		t.Fail()
	}

	// Negative: object without client.
	obj = &SObject{}
	if obj.Upsert() != nil {
		t.Fail()
	}

	// Negative: Invalid type
	obj = client.SObject("__SOME_INVALID_TYPE__").
		Set("ExternalIDField", "customExtIdField__c").
		Set("customExtIdField__c", uuid.NewString())
	if obj.Upsert() != nil {
		t.Fail()
	}

	// Negative: Invalid field
	obj = client.SObject("Case").
		Set("ExternalIDField", "customExtIdField__c").
		Set("customExtIdField__c", uuid.NewString()).
		Set("__SOME_INVALID_FIELD__", "")
	if obj.Upsert() != nil {
		t.Fail()
	}

	// Negative: Missing ext ID
	obj = client.SObject("Case").
		Set("ExternalIDField", "customExtIdField__c")
	if obj.Upsert() != nil {
		t.Fail()
	}
}

func TestSObject_Delete(t *testing.T) {
	client := requireClient(t, true)

	// Positive: create a case first then delete it and verify if it is gone.
	case1 := client.SObject("Case").
		Set("Subject", "Case created by simpleforce on "+time.Now().Format("2006/01/02 03:04:05")).
		Create().
		Get()
	if case1 == nil || case1.ID() == "" {
		t.Fatal()
	}
	caseID := case1.ID()
	if case1.Delete() != nil {
		t.Fail()
	}
	case1 = client.SObject("Case").Get(caseID)
	if case1 != nil {
		t.Fail()
	}
}

// TestSObject_GetUpdate validates updating of existing records.
func TestSObject_GetUpdate(t *testing.T) {
	client := requireClient(t, true)

	// Create a new case first.
	case1 := client.SObject("Case").
		Set("Subject", "Original").
		Create().
		Get()

	// Query the case by ID, then update the Subject.
	case2 := client.SObject("Case").
		Get(case1.ID()).
		Set("Subject", "Updated").
		Update().
		Get()

	// Query the case by ID again and check if the Subject has been updated.
	case3 := client.SObject("Case").
		Get(case2.ID())

	if case3.StringField("Subject") != "Updated" {
		t.Fail()
	}

	user1 := client.SObject("User").Create()
	log.Println(user1.ID())
}
