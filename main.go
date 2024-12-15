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

func handler(ctx context.Context,
	req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	txt := prayertexter.TextMessage{}

	if err := json.Unmarshal([]byte(req.Body), &txt); err != nil {
		slog.Error("failed to unmarshal api gateway request", "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	clnt, err := prayertexter.GetDdbClient()
	if err != nil {
		slog.Error("unmarshal failed for api gateway request", "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	txtsvc := prayertexter.FakeTextService{}

	prayertexter.MainFlow(txt, clnt, txtsvc)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Completed Successfully",
	}, nil
}

func main() {
	lambda.Start(handler)
}
