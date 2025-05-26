package test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Case struct {
	Description    string
	InitialMessage messaging.TextMessage

	ExpectedGetItemCalls    int
	ExpectedPutItemCalls    int
	ExpectedDeleteItemCalls int
	ExpectedSendTextCalls   int
	ExpectedScanCalls       int

	ExpectedMembers     []object.Member
	ExpectedPrayers     []object.Prayer
	ExpectedTexts       []messaging.TextMessage
	ExpectedPhones      object.IntercessorPhones
	ExpectedError       bool
	ExpectedPrayerQueue bool

	ExpectedExactMessageMatch []struct {
		Index   int
		Message string
	}

	ExpectedDeleteItems []struct {
		Key   string
		Table string
	}

	MockGetItemResults []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}
	MockPutItemResults []struct {
		Error error
	}
	MockDeleteItemResults []struct {
		Error error
	}
	MockSendTextResults []struct {
		Error error
	}
	MockScanResults []struct {
		Output *dynamodb.ScanOutput
		Error  error
	}
}

func SetMocks(ddbMock *mock.DDBConnecter, txtMock *mock.TextSender, tc Case) {
	ddbMock.GetItemResults = tc.MockGetItemResults
	ddbMock.PutItemResults = tc.MockPutItemResults
	ddbMock.DeleteItemResults = tc.MockDeleteItemResults
	ddbMock.ScanResults = tc.MockScanResults
	txtMock.SendTextResults = tc.MockSendTextResults
}

func RunAllCommonTests(ddbMock *mock.DDBConnecter, txtMock *mock.TextSender, t *testing.T, tc Case) {
	ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
	ValidateMembers(ddbMock.PutItemInputs, t, tc)
	ValidatePrayers(ddbMock.PutItemInputs, t, tc)
	ValidatePhones(ddbMock.PutItemInputs, t, tc)
	ValidateDeleteItem(ddbMock.DeleteItemInputs, t, tc)
	ValidateExactMessageMatch(txtMock, t, tc)
	ValidateTxtMessage(txtMock, t, tc)
}

func ValidateNumMethodCalls(ddbMock *mock.DDBConnecter, txtMock *mock.TextSender, t *testing.T, tc Case) {
	if ddbMock.GetItemCalls != tc.ExpectedGetItemCalls {
		t.Errorf("expected GetItem to be called %v, got %v",
			tc.ExpectedGetItemCalls, ddbMock.GetItemCalls)
	}

	if ddbMock.PutItemCalls != tc.ExpectedPutItemCalls {
		t.Errorf("expected PutItem to be called %v, got %v",
			tc.ExpectedPutItemCalls, ddbMock.PutItemCalls)
	}

	if ddbMock.DeleteItemCalls != tc.ExpectedDeleteItemCalls {
		t.Errorf("expected DeleteItem to be called %v, got %v",
			tc.ExpectedDeleteItemCalls, ddbMock.DeleteItemCalls)
	}

	if ddbMock.ScanCalls != tc.ExpectedScanCalls {
		t.Errorf("expected Scan to be called %v, got %v",
			tc.ExpectedScanCalls, ddbMock.ScanCalls)
	}

	if txtMock.SendTextCalls != tc.ExpectedSendTextCalls {
		t.Errorf("expected SendText to be called %v, got %v",
			tc.ExpectedSendTextCalls, txtMock.SendTextCalls)
	}
}

func ValidateMembers(inputs []dynamodb.PutItemInput, t *testing.T, tc Case) {
	index := 0

	for _, input := range inputs {
		if *input.TableName != object.DefaultMemberTable {
			continue
		}

		if index >= len(tc.ExpectedMembers) {
			t.Errorf("there are more Members in put inputs than in expected Members")
		}

		var actualMem object.Member
		if err := attributevalue.UnmarshalMap(input.Item, &actualMem); err != nil {
			t.Errorf("failed to unmarshal PutItemInput into Member: %v", err)
		}

		// Replaces date to make mocking easier.
		if actualMem.WeeklyPrayerDate != "" {
			actualMem.WeeklyPrayerDate = "dummy date/time"
		}

		expectedMem := tc.ExpectedMembers[index]
		if actualMem != expectedMem {
			t.Errorf("expected Member %v, got %v", expectedMem, actualMem)
		}

		index++
	}

	if index < len(tc.ExpectedMembers) {
		t.Errorf("there are more Members in expected Members than in put inputs")
	}
}

func ValidatePrayers(inputs []dynamodb.PutItemInput, t *testing.T, tc Case) {
	// We need to be careful here that inputs (Prayers) are not mixed with active and queued prayers, because this test
	// function cannot handle that.
	index := 0
	expectedTable := object.GetPrayerTable(tc.ExpectedPrayerQueue)

	for _, input := range inputs {
		if *input.TableName != expectedTable {
			continue
		}

		if index >= len(tc.ExpectedPrayers) {
			t.Errorf("there are more Prayers in put inputs than in expected Prayers of table type: %v", expectedTable)
		}

		var actualPryr object.Prayer
		if err := attributevalue.UnmarshalMap(input.Item, &actualPryr); err != nil {
			t.Errorf("failed to unmarshal PutItemInput into Prayer: %v", err)
		}

		// Replaces date and random ID to make mocking easier.
		if !tc.ExpectedPrayerQueue {
			actualPryr.Intercessor.WeeklyPrayerDate = "dummy date/time"
		} else {
			actualPryr.IntercessorPhone = "dummy ID"
		}

		actualPryr = replaceReminderDateIfChanged(t, actualPryr)

		expectedPryr := tc.ExpectedPrayers[index]
		if actualPryr != expectedPryr {
			t.Errorf("expected Prayer %v, got %v", expectedPryr, actualPryr)
		}

		index++
	}

	if index < len(tc.ExpectedPrayers) {
		t.Errorf("there are more Prayers in expected Prayers than in put inputs of table type: %v", expectedTable)
	}
}

// replaceReminderDateIfChanged replaces date with a testable "date changed" string only if the date is within 1 minute
// of time.Now(). This at least shows that the date was updated and allows us to test whether it was changed or not.
func replaceReminderDateIfChanged(t *testing.T, actualPryr object.Prayer) object.Prayer {
	if actualPryr.ReminderDate != "" {
		currentDate := time.Now()
		reminderDate, err := time.Parse(time.RFC3339, actualPryr.ReminderDate)
		if err != nil {
			t.Errorf("unexpected error parsing time: %v", err)
		}
		diffMins := currentDate.Sub(reminderDate).Minutes()
		if diffMins < 1 {
			actualPryr.ReminderDate = "date changed"
		}
	}

	return actualPryr
}

func ValidatePhones(inputs []dynamodb.PutItemInput, t *testing.T, tc Case) {
	index := 0

	for _, input := range inputs {
		if *input.TableName != object.DefaultIntercessorPhonesTable {
			continue
		} else if val, ok := input.Item[object.IntercessorPhonesKey]; !ok {
			continue
		} else if stringVal, isString := val.(*types.AttributeValueMemberS); !isString {
			continue
		} else if stringVal.Value != object.IntercessorPhonesKeyValue {
			continue
		}

		if index > 1 {
			t.Errorf("there are more IntercessorPhones in expected IntercessorPhones than 1 which is not expected")
		}

		var actualPhones object.IntercessorPhones
		if err := attributevalue.UnmarshalMap(input.Item, &actualPhones); err != nil {
			t.Errorf("failed to unmarshal PutItemInput into IntercessorPhones: %v", err)
		}

		if !reflect.DeepEqual(actualPhones, tc.ExpectedPhones) {
			t.Errorf("expected IntercessorPhones %v, got %v", tc.ExpectedPhones, actualPhones)
		}

		index++
	}
}

func ValidateDeleteItem(inputs []dynamodb.DeleteItemInput, t *testing.T, tc Case) {
	index := 0

	for _, input := range inputs {
		if index >= len(tc.ExpectedDeleteItems) {
			t.Errorf("there are more delete item inputs than expected delete items")
		}

		switch *input.TableName {
		case object.DefaultMemberTable:
			testDeleteMember(input, &index, t, tc)
		case object.DefaultActivePrayersTable, object.DefaultQueuedPrayersTable:
			testDeletePrayer(input, &index, t, tc)
		default:
			t.Errorf("unexpected table name, got %v", *input.TableName)
		}
	}

	if index < len(tc.ExpectedDeleteItems) {
		t.Errorf("there are more expected delete items than delete item inputs")
	}
}

func testDeleteMember(input dynamodb.DeleteItemInput, index *int, t *testing.T, tc Case) {
	if *input.TableName != tc.ExpectedDeleteItems[*index].Table {
		t.Errorf("expected Member table %v, got %v",
			tc.ExpectedDeleteItems[*index].Table, *input.TableName)
	}

	mem := object.Member{}
	if err := attributevalue.UnmarshalMap(input.Key, &mem); err != nil {
		t.Fatalf("failed to unmarshal to Member: %v", err)
	}

	if mem.Phone != tc.ExpectedDeleteItems[*index].Key {
		t.Errorf("expected Member phone %v for delete key, got %v",
			tc.ExpectedDeleteItems[*index].Key, mem.Phone)
	}

	*index++
}

func testDeletePrayer(input dynamodb.DeleteItemInput, index *int, t *testing.T, tc Case) {
	if *input.TableName != tc.ExpectedDeleteItems[*index].Table {
		t.Errorf("expected Prayer table %v, got %v",
			tc.ExpectedDeleteItems[*index].Table, *input.TableName)
	}

	pryr := object.Prayer{}
	if err := attributevalue.UnmarshalMap(input.Key, &pryr); err != nil {
		t.Fatalf("failed to unmarshal to Prayer: %v", err)
	}

	if pryr.IntercessorPhone != tc.ExpectedDeleteItems[*index].Key {
		t.Errorf("expected Prayer phone %v for delete key, got %v",
			tc.ExpectedDeleteItems[*index].Key, pryr.IntercessorPhone)
	}

	*index++
}

func ValidateTxtMessage(txtMock *mock.TextSender, t *testing.T, tc Case) {
	index := 0

	for _, input := range txtMock.SendTextInputs {
		if index >= len(tc.ExpectedTexts) {
			t.Errorf("there are more text message inputs than expected texts")
		}

		// Some text messages use PLACEHOLDER and replace that with the txt recipients name. Therefor to make testing
		// easier, the message body is replaced by the msg constant.
		switch {
		case strings.Contains(*input.MessageBody, "Hello! Please pray for"):
			input.MessageBody = aws.String(messaging.MsgPrayerIntro)
		case strings.Contains(*input.MessageBody, "There was profanity found in your message"):
			input.MessageBody = aws.String(messaging.MsgProfanityDetected)
		case strings.Contains(*input.MessageBody, "You're prayer request has been prayed for by"):
			input.MessageBody = aws.String(messaging.MsgPrayerConfirmation)
		case strings.Contains(*input.MessageBody, "This is a friendly reminder to pray for"):
			input.MessageBody = aws.String(messaging.MsgPrayerReminder)
		}

		receivedText := messaging.TextMessage{
			Body:  *input.MessageBody,
			Phone: *input.DestinationPhoneNumber,
		}

		// This part makes mocking messages less painful. We do not need to worry about new lines, pre, or post
		// messages. They are removed when messages are tested.
		for _, t := range []*messaging.TextMessage{&receivedText, &tc.ExpectedTexts[index]} {
			for _, str := range []string{"\n", messaging.MsgPre, messaging.MsgPost} {
				t.Body = strings.ReplaceAll(t.Body, str, "")
			}
		}

		if receivedText != tc.ExpectedTexts[index] {
			t.Errorf("expected txt %v, got %v", tc.ExpectedTexts[index], receivedText)
		}

		index++
	}

	if index < len(tc.ExpectedTexts) {
		t.Errorf("there are more expected texts than text message inputs")
	}
}

func ValidateExactMessageMatch(txtMock *mock.TextSender, t *testing.T, tc Case) {
	if len(tc.ExpectedExactMessageMatch) == 0 {
		return
	}

	for _, match := range tc.ExpectedExactMessageMatch {
		if *txtMock.SendTextInputs[match.Index].MessageBody != match.Message {
			t.Errorf("expected message %v, got %v", match.Message, *txtMock.SendTextInputs[match.Index].MessageBody)
		}
	}
}
