/*
Announcer is a helper application for the main application prayertexter. Announcer can be used to send announcements
to all prayertexter members. This could be for general updates, taking down or turning up prayertexter, or alerts for
things such as outages or service restoration.
*/
package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/4JesusApps/prayertexter/internal/buildinfo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

//nolint:revive // IGNORING UNUSED CTX AND REQ VARIABLES FOR NOW; REMOVE ONCE THIS FUNCTION IS IMPLEMENTED
func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	slog.InfoContext(ctx, "running announcer", "version", buildinfo.Version())
	// Place holder for future code.
	// Don't forget to remove the above nolint when this is implemented.

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "Success"}, nil
}

func main() {
	lambda.Start(handler)
}
