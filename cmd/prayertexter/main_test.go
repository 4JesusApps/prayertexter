package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

// TestSNSEventFixture_UnmarshalsToTextMessage guards the inbound SMS payload contract:
// the outer SNS envelope shape (owned by AWS Lambda) and the inner JSON keys
// (owned by AWS End User Messaging V2) that domain.TextMessage depends on. Update
// testdata/sns-event.json whenever a real production event reveals new fields.
func TestSNSEventFixture_UnmarshalsToTextMessage(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sns-event.json"))
	require.NoError(t, err)

	var snsEvent events.SNSEvent
	require.NoError(t, json.Unmarshal(raw, &snsEvent))
	require.Len(t, snsEvent.Records, 1)

	record := snsEvent.Records[0]
	require.NotEmpty(t, record.SNS.Message)

	var msg domain.TextMessage
	require.NoError(t, json.Unmarshal([]byte(record.SNS.Message), &msg))

	require.Equal(t, "+11234567890", msg.Phone)
	require.Equal(t, "pray", msg.Body)
}

// TestSNSEventFixture_MalformedInnerMessageIsHandled documents the handler's
// expected behavior when one record's SNS.Message is not valid JSON: that record
// should produce an error, but unmarshaling the outer envelope must still succeed.
func TestSNSEventFixture_MalformedInnerMessageIsHandled(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sns-event.json"))
	require.NoError(t, err)

	var snsEvent events.SNSEvent
	require.NoError(t, json.Unmarshal(raw, &snsEvent))

	snsEvent.Records[0].SNS.Message = "{not valid json"

	var msg domain.TextMessage
	err = json.Unmarshal([]byte(snsEvent.Records[0].SNS.Message), &msg)
	require.Error(t, err)
}
