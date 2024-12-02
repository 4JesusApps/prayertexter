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

const (
	intercessorsTable = "Intercessors"
	membersTable      = "Members"
)

type TextMessage struct {
	Body        string `json:"body"`
	PhoneNumber string `json:"phone-number"`
}

func sendText(body string, recipient string) {
	log.Printf("Sending to: %v\n", recipient)
	log.Printf("Body: %v\n", body)
}

func signUp(txt TextMessage, p Person) {
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
		p.SetupStatus = "in-progress"
		p.SetupStage = 1
		p.put(membersTable)
		p.sendMessage(nameRequest)
	} else if txt.Body != "2" && p.SetupStage == 1 {
		// stage 2 name request
		p.SetupStage = 2
		p.Name = txt.Body
		p.put(membersTable)
		p.sendMessage(memberType)
	} else if txt.Body == "2" && p.SetupStage == 1 {
		// stage 2 name request
		p.SetupStage = 2
		p.Name = "Anonymous"
		p.put(membersTable)
		p.sendMessage(memberType)
	} else if txt.Body == "1" && p.SetupStage == 2 {
		// final message for member sign up
		p.SetupStatus = "completed"
		p.SetupStage = 99
		p.put(membersTable)
		p.sendMessage(prayerRequestInstructions)
	} else if txt.Body == "2" && p.SetupStage == 2 {
		// stage 3 intercessor sign up
		p.SetupStage = 3
		p.put(membersTable)
		p.put(intercessorsTable)
		p.sendMessage(prayerNumRequest)
	} else if p.SetupStage == 3 {
		// final message for intercessor sign up
		if num, err := strconv.Atoi(txt.Body); err == nil {
			p.SetupStatus = "completed"
			p.SetupStage = 99
			p.PrayerLimit = num
			p.put(membersTable)
			p.put(intercessorsTable)
			p.sendMessage(intercessorInstructions)
		} else {
			p.sendMessage(wrongInput)
		}
	} else {
		// catch all response for incorrect input
		p.sendMessage(wrongInput)
	}
}

func prayerRequest(txt TextMessage, p Person) {
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
			Requestor:        p,
		}
		pryr.put()
		sendText(prayerIntro+pryr.Request, i.Phone)
	}

	sendText(prayerConfirmation, p.Phone)
}

func mainFlow(txt TextMessage) error {
	const (
		removeUser = "You have been removed from prayer texter. If you ever want to sign back up, text the word pray to this number."
	)

	p := Person{
		Phone: txt.PhoneNumber,
	}

	p = p.get(membersTable)

	if strings.ToLower(txt.Body) == "pray" || p.SetupStatus == "in-progress" {
		signUp(txt, p)
	} else if strings.ToLower(txt.Body) == "cancel" || strings.ToLower(txt.Body) == "stop" {
		p.delete()
		p.sendMessage(removeUser)
	} else if p.SetupStatus == "completed" {
		prayerRequest(txt, p)
	} else if p.SetupStatus == "" {
		log.Printf("%v is not a registered user, dropping message", p.Phone)
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
