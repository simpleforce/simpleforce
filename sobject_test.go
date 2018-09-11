package simpleforce

import (
	"log"
	"testing"
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
	meta := *client.SObject("Case").Describe()
	if meta == nil {
		t.Fail()
	} else {
		if meta["name"].(string) != "Case" {
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
		t.Fail()
		return
	}
	if queryResult.TotalSize < 1 {
		t.Fail()
		return
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
