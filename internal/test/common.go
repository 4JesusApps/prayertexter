package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/4JesusApps/prayertexter/internal/test/testutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Case struct {
	Description    string
	InitialMessage messaging.TextMessage

	ExpectedGetItemCalls    int
	ExpectedPutItemCalls    int
	ExpectedDeleteItemCalls int
	ExpectedSendTextCalls   int
	ExpectedScanCalls       int

	ExpectedMembers     []model.Member
	ExpectedPrayers     []model.Prayer
	ExpectedTexts       []messaging.TextMessage
	ExpectedBlockPhones model.BlockedPhones
	ExpectedIntPhones   model.IntercessorPhones
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

	MockGetItemResults []testutil.GetItemResult
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
	assert.Equal(t, tc.ExpectedGetItemCalls, ddbMock.GetItemCalls, "GetItem call count")
	assert.Equal(t, tc.ExpectedPutItemCalls, ddbMock.PutItemCalls, "PutItem call count")
	assert.Equal(t, tc.ExpectedDeleteItemCalls, ddbMock.DeleteItemCalls, "DeleteItem call count")
	assert.Equal(t, tc.ExpectedScanCalls, ddbMock.ScanCalls, "Scan call count")
	assert.Equal(t, tc.ExpectedSendTextCalls, txtMock.SendTextCalls, "SendText call count")
}

func ValidateMembers(inputs []dynamodb.PutItemInput, t *testing.T, tc Case) {
	index := 0

	for _, input := range inputs {
		if *input.TableName != "Member" {
			continue
		}

		require.Less(t, index, len(tc.ExpectedMembers),
			"there are more Members in put inputs than in expected Members")

		var actualMem model.Member
		err := attributevalue.UnmarshalMap(input.Item, &actualMem)
		require.NoError(t, err, "failed to unmarshal PutItemInput into Member")

		// Replaces date to make mocking easier.
		if actualMem.WeeklyPrayerDate != "" {
			actualMem.WeeklyPrayerDate = "dummy date/time"
		}

		expectedMem := tc.ExpectedMembers[index]
		assert.Equal(t, expectedMem, actualMem, "Member mismatch")

		index++
	}

	assert.Equal(t, len(tc.ExpectedMembers), index,
		"there are more Members in expected Members than in put inputs")
}

func ValidatePrayers(inputs []dynamodb.PutItemInput, t *testing.T, tc Case) {
	// We need to be careful here that inputs (Prayers) are not mixed with active and queued prayers, because this test
	// function cannot handle that.
	index := 0
	expectedTable := "ActivePrayer"
	if tc.ExpectedPrayerQueue {
		expectedTable = "QueuedPrayer"
	}

	for _, input := range inputs {
		if *input.TableName != expectedTable {
			continue
		}

		require.Less(t, index, len(tc.ExpectedPrayers),
			"there are more Prayers in put inputs than in expected Prayers of table type: %v", expectedTable)

		var actualPryr model.Prayer
		err := attributevalue.UnmarshalMap(input.Item, &actualPryr)
		require.NoError(t, err, "failed to unmarshal PutItemInput into Prayer")

		// Replaces date and random ID to make mocking easier.
		if !tc.ExpectedPrayerQueue {
			actualPryr.Intercessor.WeeklyPrayerDate = "dummy date/time"
		} else {
			actualPryr.IntercessorPhone = "dummy ID"
		}

		actualPryr = replaceReminderDateIfChanged(t, actualPryr)

		expectedPryr := tc.ExpectedPrayers[index]
		assert.Equal(t, expectedPryr, actualPryr, "Prayer mismatch")

		index++
	}

	assert.Equal(t, len(tc.ExpectedPrayers), index,
		"there are more Prayers in expected Prayers than in put inputs of table type: %v", expectedTable)
}

// replaceReminderDateIfChanged replaces date with a testable "date changed" string only if the date is within 1 minute
// of time.Now(). This at least shows that the date was updated and allows us to test whether it was changed or not.
func replaceReminderDateIfChanged(t *testing.T, actualPryr model.Prayer) model.Prayer {
	if actualPryr.ReminderDate != "" {
		currentDate := time.Now()
		reminderDate, err := time.Parse(time.RFC3339, actualPryr.ReminderDate)
		require.NoError(t, err, "unexpected error parsing time")
		diffMins := currentDate.Sub(reminderDate).Minutes()
		if diffMins < 1 {
			actualPryr.ReminderDate = "date changed"
		}
	}

	return actualPryr
}

func ValidatePhones(inputs []dynamodb.PutItemInput, t *testing.T, tc Case) {
	intIndex := 0
	blockIndex := 0

	for _, input := range inputs {
		// Check if this is a phone-related table (both IntercessorPhones and BlockedPhones use the same table)
		if *input.TableName != "General" {
			continue
		}

		// Check if this item has the correct key field
		val, ok := input.Item[model.IntercessorPhonesKey]
		if !ok {
			continue
		}

		stringVal, isString := val.(*types.AttributeValueMemberS)
		if !isString {
			continue
		}

		// Handle IntercessorPhones
		if stringVal.Value == model.IntercessorPhonesKeyValue {
			validatePhoneType(input, tc.ExpectedIntPhones, &intIndex, t)
		}

		// Handle BlockedPhones
		if stringVal.Value == model.BlockedPhonesKeyValue {
			validatePhoneType(input, tc.ExpectedBlockPhones, &blockIndex, t)
		}
	}
}

func validatePhoneType[T any](input dynamodb.PutItemInput, expected T, index *int, t *testing.T) {
	typeName := fmt.Sprintf("%T", expected)

	assert.LessOrEqual(t, *index, 0,
		"there are more %s in put inputs than 1 which is not expected", typeName)

	var actual T
	err := attributevalue.UnmarshalMap(input.Item, &actual)
	require.NoError(t, err, "failed to unmarshal PutItemInput into %s", typeName)

	assert.Equal(t, expected, actual, "%s mismatch", typeName)

	*index++
}

func ValidateDeleteItem(inputs []dynamodb.DeleteItemInput, t *testing.T, tc Case) {
	index := 0

	for _, input := range inputs {
		require.Less(t, index, len(tc.ExpectedDeleteItems),
			"there are more delete item inputs than expected delete items")

		switch *input.TableName {
		case "Member":
			testDeleteMember(input, &index, t, tc)
		case "ActivePrayer", "QueuedPrayer":
			testDeletePrayer(input, &index, t, tc)
		default:
			assert.Fail(t, "unexpected table name", "got %v", *input.TableName)
		}
	}

	assert.Equal(t, len(tc.ExpectedDeleteItems), index,
		"there are more expected delete items than delete item inputs")
}

func testDeleteMember(input dynamodb.DeleteItemInput, index *int, t *testing.T, tc Case) {
	assert.Equal(t, tc.ExpectedDeleteItems[*index].Table, *input.TableName, "Member table name")

	mem := model.Member{}
	err := attributevalue.UnmarshalMap(input.Key, &mem)
	require.NoError(t, err, "failed to unmarshal to Member")

	assert.Equal(t, tc.ExpectedDeleteItems[*index].Key, mem.Phone, "Member phone for delete key")

	*index++
}

func testDeletePrayer(input dynamodb.DeleteItemInput, index *int, t *testing.T, tc Case) {
	assert.Equal(t, tc.ExpectedDeleteItems[*index].Table, *input.TableName, "Prayer table name")

	pryr := model.Prayer{}
	err := attributevalue.UnmarshalMap(input.Key, &pryr)
	require.NoError(t, err, "failed to unmarshal to Prayer")

	assert.Equal(t, tc.ExpectedDeleteItems[*index].Key, pryr.IntercessorPhone, "Prayer phone for delete key")

	*index++
}

func ValidateTxtMessage(txtMock *mock.TextSender, t *testing.T, tc Case) {
	index := 0

	for _, input := range txtMock.SendTextInputs {
		require.Less(t, index, len(tc.ExpectedTexts),
			"there are more text message inputs than expected texts")

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
		for _, txt := range []*messaging.TextMessage{&receivedText, &tc.ExpectedTexts[index]} {
			for _, str := range []string{"\n", messaging.MsgPre, messaging.MsgPost} {
				txt.Body = strings.ReplaceAll(txt.Body, str, "")
			}
		}

		assert.Equal(t, tc.ExpectedTexts[index], receivedText, "text message mismatch")

		index++
	}

	assert.Equal(t, len(tc.ExpectedTexts), index,
		"there are more expected texts than text message inputs")
}

func ValidateExactMessageMatch(txtMock *mock.TextSender, t *testing.T, tc Case) {
	if len(tc.ExpectedExactMessageMatch) == 0 {
		return
	}

	for _, match := range tc.ExpectedExactMessageMatch {
		assert.Equal(t, match.Message, *txtMock.SendTextInputs[match.Index].MessageBody,
			"exact message match at index %d", match.Index)
	}
}
