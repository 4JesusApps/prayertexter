/*
Prayertexter is a prayer application over text messaging. Users can sign up over text message and become members. There
are 2 types of members: regular and intercessors. Regular members are able to send in prayer requests to the
prayertexter phone number. Intercessors are able to do the same, as well as receive prayer requests sent by regular
members. When an intercessor receives a prayer request, they must pray over that request in a reasonable amount of time
and text back the word "prayed" to prayertexter. The prayer requestor will then be alerted that their request was prayed
for.

To sign up over text message, one must text the word "pray" to the prayertexter phone number. There is a sign up flow
in which the user will text back and forth between prayertexter until they are officially signed up and in the
prayertexter system. Sign up options include whether they want to remain anonymous, to be an intercessor, and how many
prayers they are willing to receive per week (if they agreed to be an intercessor).
*/
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/prayertexter"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// MUST BE SET by go build -ldflags "-X main.version=999"
// like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty

//lint:ignore U1000 - var used in Makefile
var version string // do not remove or modify

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	msg := messaging.TextMessage{}

	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to unmarshal api gateway request", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	ddbClnt, err := db.GetDdbClient(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to get dynamodb client", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	smsClnt, err := messaging.GetSmsClient(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to get sms client", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	if err := prayertexter.MainFlow(ctx, ddbClnt, smsClnt, msg); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "Success"}, nil
}

func main() {
	lambda.Start(handler)
}
