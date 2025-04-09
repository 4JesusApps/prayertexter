/*
Stateresolver is a helper application for the main application prayertexter. Stateresolver runs on a scheduled basis
similar to a cronjob. It attempts to assign prayers in the prayer queue. It also checks all active prayers and sends a
reminder text message to the assigned intercessor that the prayer has not been prayed for for a configurable amount of
time to prevent prayer requests from getting stale/forgotten. It also performs some level of retry/recovery mechanisms
for previously failed operations.
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
