package handlers

import (
	"testing"
)

func TestEmptyTicketPriceJSON(t *testing.T) {
	testStr := `{
		"ValidateMessagesShowId":"_validatorMessage",
		"status":true,"httpstatus":200,
		"data":{"OT":[],"train_no":"760000D63803"},
		"messages":[],"validateMessages":{},
		"updatetime":1515424867
		}`

	testJSON, _ := verifyTicketPrice(&testStr)
	if !emptyTicketPriceJSON(testJSON) {
		t.Error("Empty ticket price json considered non empty")
	}

	testStr = `{
		"ValidateMessagesShowId":"_validatorMessage",
		"status":true,"httpstatus":200,
		"data":{"OT":[],"train_no":"760000D63803","M":"¥933.0"},
		"messages":[],"validateMessages":{}
		}
	`
	testJSON, _ = verifyTicketPrice(&testStr)
	if emptyTicketPriceJSON(testJSON) {
		t.Error("Non-empty ticket price json considered empty")
	}
}
