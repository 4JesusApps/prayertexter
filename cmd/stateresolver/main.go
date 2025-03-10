package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// MUST BE SET by go build -ldflags "-X main.version=999"
// like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty

//lint:ignore U1000 - var used in Makefile
var version string // do not remove or modify

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// place holder for future code

	return events.APIGatewayProxyResponse{StatusCode: 200, Body: "Success"}, nil
}

func main() {
	lambda.Start(handler)
}
