package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	prayertexter "prayertexter/app"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// MUST BE SET by go build -ldflags "-X main.version=999"
// like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty

//lint:ignore U1000 - var used in Makefile
var version string // do not remove or modify

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	msg := prayertexter.TextMessage{}

	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		slog.Error("lambda handler: failed to unmarshal api gateway request", "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	ddbClnt, err := prayertexter.GetDdbClient()
	if err != nil {
		slog.Error("lambda handler: failed to get dynamodb client", "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	smsClnt, err := prayertexter.GetSmsClient()
	if err != nil {
		slog.Error("lambda handler: failed to get sms client", "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	if err := prayertexter.MainFlow(msg, ddbClnt, smsClnt); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: 200, Body: "Success"}, nil
}

func main() {
	lambda.Start(handler)
}
