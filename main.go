package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// MUST BE SET by go build -ldflags "-X main.version=999"
// like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty

//lint:ignore U1000 - var used in Makefile
var version string // do not remove or modify

type TextMessage struct {
	Body        string `json:"body"`
	PhoneNumber string `json:"phone-number"`
}

func sendText(body string, recipient string) {
	log.Printf("Sending to: %v\n", recipient)
	log.Printf("Body: %v\n", body)
}

func signUp(txt TextMessage, per Person) {
	const (
		nameRequest               = "Text your name, or 2 to stay anonymous"
		memberType                = "Text 1 for prayer request, or 2 to be added to the intercessors list (to pray for others)"
		prayerRequestInstructions = "You are now signed up to send prayer requests! Please send them directly to this number."
		prayerNumRequest          = "Send the max number of prayer texts you are willing to receive and pray for per week."
		intercessorInstructions   = "You are now signed up to receive prayer requests. Please try to pray for the requests ASAP. Once you are done praying, send 'prayed' back to this number for confirmation."
		wrongInput                = "Wrong input received during sign up process. Please try again."
	)

	if strings.ToLower(txt.Body) == "pray" {
		// stage 1
		per.SetupStatus = "in-progress"
		per.SetupStage = "1"
		per.put("Members")
		per.sendMessage(nameRequest)
	} else if txt.Body != "2" && per.SetupStage == "1" {
		// stage 2 name request
		per.SetupStage = "2"
		per.Name = txt.Body
		per.put("Members")
		per.sendMessage(memberType)
	} else if txt.Body == "2" && per.SetupStage == "1" {
		// stage 2 name request
		per.SetupStage = "2"
		per.Name = "Anonymous"
		per.put("Members")
		per.sendMessage(memberType)
	} else if txt.Body == "1" && per.SetupStage == "2" {
		// final message for member sign up
		per.SetupStatus = "completed"
		per.SetupStage = "99"
		per.put("Members")
		per.sendMessage(prayerRequestInstructions)
	} else if txt.Body == "2" && per.SetupStage == "2" {
		// stage 3 intercessor sign up
		per.SetupStage = "3"
		per.put("Members")
		per.put("Intercessors")
		per.sendMessage(prayerNumRequest)
	} else if per.SetupStage == "3" {
		// final message for intercessor sign up
		if _, err := strconv.Atoi(txt.Body); err == nil {
			per.SetupStatus = "completed"
			per.SetupStage = "99"
			per.PrayerLimit = txt.Body
			per.put("Members")
			per.put("Intercessors")
			per.sendMessage(intercessorInstructions)
		} else {
			per.sendMessage(wrongInput)
		}
	} else {
		// catch all response for incorrect input
		per.sendMessage(wrongInput)
	}
}

func intercessorSelector() {
	
}

func prayerRequest(txt TextMessage, per Person) {
	const (
		prayerIntro        = "Hello! Please pray for this person:\n"
		prayerConfirmation = "Your prayer request has been sent out!"
	)

	int1 := Person{
		Name:  "Person 1",
		Phone: "111-111-1111",
	}
	int2 := Person{
		Name:  "Person 2",
		Phone: "222-222-2222",
	}
	int3 := Person{
		Name:  "Person 3",
		Phone: "333-333-3333",
	}

	for _, i := range []Person{int1, int2, int3} {
		pryr := Prayer{
			Intercessor:      i,
			IntercessorPhone: i.Phone,
			Request:          txt.Body,
			Requestor:        per,
		}
		pryr.put()
		sendText(prayerIntro+pryr.Request, i.Phone)
	}

	sendText(prayerConfirmation, per.Phone)
}

func mainFlow(txt TextMessage) error {
	const (
		removeUser = "You have been removed from prayer texter. If you ever want to sign back up, text the word pray to this number."
	)

	per := Person{
		Phone: txt.PhoneNumber,
	}

	per = per.get("Members")

	if strings.ToLower(txt.Body) == "pray" || per.SetupStatus == "in-progress" {
		signUp(txt, per)
	} else if strings.ToLower(txt.Body) == "cancel" || strings.ToLower(txt.Body) == "stop" {
		per.delete()
		per.sendMessage(removeUser)
	} else if per.SetupStatus == "completed" {
		prayerRequest(txt, per)
	} else if per.SetupStatus == "" {
		log.Printf("%v is not a registered user, dropping message", per.Phone)
	}

	return nil
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (
	events.APIGatewayProxyResponse, error) {
	txt := TextMessage{}

	err := json.Unmarshal([]byte(req.Body), &txt)
	if err != nil {
		log.Fatalf("failed to unmarshal api gateway request, %v", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	err = mainFlow(txt)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Completed Successfully",
	}, nil
}

func main() {
	lambda.Start(handler)
}
