/*
This file is used for sam local testing only. It uses an api gateway trigger which is easier to set and up with sam
local.
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

// MUST BE SET by go build -ldflags "-X main.version=999" like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty.
var version string // do not remove or modify

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	slog.InfoContext(ctx, "running prayertexter", "version", version)
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

	if err = prayertexter.MainFlow(ctx, ddbClnt, smsClnt, msg); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "Success\n"}, nil
}

func main() {
	lambda.Start(handler)
}
