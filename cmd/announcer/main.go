/*
Announcer is a helper application for the main application prayertexter. Announcer can be used to send announcements
to all prayertexter members. This could be for general updates, taking down or turning up prayertexter, or alerts for
things such as outages or service restoration.
*/
package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// MUST BE SET by go build -ldflags "-X main.version=999"
// like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty

//lint:ignore U1000 - var used in Makefile
var version string // do not remove or modify

//nolint:revive // IGNORING UNUSED CTX AND REQ VARIABLES FOR NOW; REMOVE ONCE THIS FUNCTION IS IMPLEMENTED
func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Place holder for future code.
	// Don't forget to remove the above nolint when this is implemented.

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "Success"}, nil
}

func main() {
	lambda.Start(handler)
}
