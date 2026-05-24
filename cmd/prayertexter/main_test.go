package main

import (
	"context"
	"encoding/json"
	"errors"
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

// stubHandler is a test double for the messageHandler interface that records
// every Handle invocation and can be configured to return an error per call.
type stubHandler struct {
	calls []domain.TextMessage
	// errs is consumed in order; if nil or exhausted, Handle returns nil.
	errs []error
}

func (s *stubHandler) Handle(_ context.Context, msg domain.TextMessage) error {
	s.calls = append(s.calls, msg)
	if len(s.errs) == 0 {
		return nil
	}
	err := s.errs[0]
	s.errs = s.errs[1:]
	return err
}

// snsRecord builds a single SNS event record whose Message body is the JSON
// encoding of a TextMessage. Used to assemble realistic batched events.
func snsRecord(messageID, phone, body string) events.SNSEventRecord {
	inner, _ := json.Marshal(domain.TextMessage{Phone: phone, Body: body})
	return events.SNSEventRecord{
		SNS: events.SNSEntity{
			MessageID: messageID,
			Message:   string(inner),
		},
	}
}

// TestProcessRecords_SingleValidRecord covers the happy path: a well-formed
// SNS record is parsed and dispatched to the handler exactly once.
func TestProcessRecords_SingleValidRecord(t *testing.T) {
	h := &stubHandler{}
	records := []events.SNSEventRecord{snsRecord("m1", "+11234567890", "pray")}

	err := processRecords(context.Background(), h, records)

	require.NoError(t, err)
	require.Len(t, h.calls, 1)
	require.Equal(t, domain.TextMessage{Phone: "+11234567890", Body: "pray"}, h.calls[0])
}

// TestProcessRecords_FromFixture wires the captured production payload through
// processRecords end-to-end, proving the fixture contract and the dispatch
// path agree on field names.
func TestProcessRecords_FromFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sns-event.json"))
	require.NoError(t, err)

	var snsEvent events.SNSEvent
	require.NoError(t, json.Unmarshal(raw, &snsEvent))

	h := &stubHandler{}
	require.NoError(t, processRecords(context.Background(), h, snsEvent.Records))
	require.Len(t, h.calls, 1)
	require.Equal(t, "+11234567890", h.calls[0].Phone)
	require.Equal(t, "pray", h.calls[0].Body)
}

// TestProcessRecords_EmptyRecords asserts the contract that an SNS event with
// no records is treated as a hard error (Lambda misconfiguration, not a
// retryable per-record failure).
func TestProcessRecords_EmptyRecords(t *testing.T) {
	h := &stubHandler{}

	err := processRecords(context.Background(), h, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no records")
	require.Empty(t, h.calls)
}

// TestProcessRecords_MultipleRecords exercises the batched-SNS warning path
// and confirms each record is dispatched independently.
func TestProcessRecords_MultipleRecords(t *testing.T) {
	h := &stubHandler{}
	records := []events.SNSEventRecord{
		snsRecord("m1", "+11111111111", "pray"),
		snsRecord("m2", "+12222222222", "help"),
		snsRecord("m3", "+13333333333", "prayed"),
	}

	err := processRecords(context.Background(), h, records)

	require.NoError(t, err)
	require.Len(t, h.calls, 3)
	require.Equal(t, "+11111111111", h.calls[0].Phone)
	require.Equal(t, "+12222222222", h.calls[1].Phone)
	require.Equal(t, "+13333333333", h.calls[2].Phone)
}

// TestProcessRecords_MalformedRecordContinues asserts the partial-failure
// contract: a single bad record produces a joined error but does not block
// the remaining records from being processed.
func TestProcessRecords_MalformedRecordContinues(t *testing.T) {
	h := &stubHandler{}
	bad := events.SNSEventRecord{
		SNS: events.SNSEntity{MessageID: "bad", Message: "{not valid json"},
	}
	records := []events.SNSEventRecord{
		snsRecord("m1", "+11111111111", "pray"),
		bad,
		snsRecord("m3", "+13333333333", "help"),
	}

	err := processRecords(context.Background(), h, records)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal sns record 1 (bad)")
	require.Len(t, h.calls, 2)
	require.Equal(t, "+11111111111", h.calls[0].Phone)
	require.Equal(t, "+13333333333", h.calls[1].Phone)
}

// TestProcessRecords_HandlerErrorJoined confirms that Handle errors from
// individual records are joined via errors.Join so callers can see every
// failure rather than only the first.
func TestProcessRecords_HandlerErrorJoined(t *testing.T) {
	handleErr1 := errors.New("router boom 1")
	handleErr2 := errors.New("router boom 2")
	h := &stubHandler{errs: []error{handleErr1, nil, handleErr2}}
	records := []events.SNSEventRecord{
		snsRecord("m1", "+11111111111", "pray"),
		snsRecord("m2", "+12222222222", "help"),
		snsRecord("m3", "+13333333333", "prayed"),
	}

	err := processRecords(context.Background(), h, records)

	require.Error(t, err)
	require.ErrorIs(t, err, handleErr1)
	require.ErrorIs(t, err, handleErr2)
	require.Contains(t, err.Error(), "failed to process sns record 0 (m1)")
	require.Contains(t, err.Error(), "failed to process sns record 2 (m3)")
	require.Len(t, h.calls, 3)
}
