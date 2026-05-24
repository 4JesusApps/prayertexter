package main

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

// TestParseRequest_Valid covers the happy path: a curl-style JSON body
// matching the production SNS inner-message shape decodes into TextMessage.
// This is the contract dev/ scripts depend on when posting to sam local.
func TestParseRequest_Valid(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		Body: `{"originationNumber":"+11234567890","messageBody":"pray"}`,
	}

	msg, err := parseRequest(req)

	require.NoError(t, err)
	require.Equal(t, domain.TextMessage{Phone: "+11234567890", Body: "pray"}, msg)
}

// TestParseRequest_Malformed ensures parse errors surface to the caller so
// the handler can return a 500 instead of dispatching a zero-value message.
func TestParseRequest_Malformed(t *testing.T) {
	req := events.APIGatewayProxyRequest{Body: "{not json"}

	_, err := parseRequest(req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "parse api gateway request")
}

// TestParseRequest_EmptyBody asserts that an empty body is rejected at parse
// time. Without this, a downstream router would silently process a phone-less
// message and fail deeper in the stack with a less actionable error.
func TestParseRequest_EmptyBody(t *testing.T) {
	req := events.APIGatewayProxyRequest{Body: ""}

	_, err := parseRequest(req)

	require.Error(t, err)
}
