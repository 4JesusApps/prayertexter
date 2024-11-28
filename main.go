package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// MUST BE SET by go build -ldflags "-X main.version=999"
// like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty

//lint:ignore U1000 - var used in Makefile
var version string // do not remove or modify

const (
	prayerIntro        = "Hello! Please pray for this person:"
	prayerConfirmation = "Your prayer request has been sent out!"
)

type TextMessage struct {
	Body        string `json:"body"`
	PhoneNumber string `json:"phone-number"`
}

type Person struct {
	Name        string
	PhoneNumber string
	PrayerLimit string
	SetupStage  string
	SetupStatus string
}

type Prayer struct {
	People      []string
	PhoneNumber string
	Request     string
}

func (per Person) sendMessage(body string) {
	sendText(body, per.PhoneNumber)
}

func sendText(body string, recipient string) {
	log.Printf("Sending to: %v\n", recipient)
	log.Printf("Body: %v\n", body)
}

func getDdbClient() *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	local, err := strconv.ParseBool(os.Getenv("AWS_SAM_LOCAL"))

	if err != nil {
		log.Fatalf("unable to convert string to boolean, %v", err)
	}

	var client *dynamodb.Client

	if local {
		client = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String("http://dynamodb:8000")
		})
	} else {
		client = dynamodb.NewFromConfig(cfg)
	}

	return client
}

func (per Person) get(table string) Person {
	resp := getItem(per.PhoneNumber, table)

	err := attributevalue.UnmarshalMap(resp.Item, &per)
	if err != nil {
		log.Fatalf("unmarshal failed, %v", err)
	}

	return per
}

func (p Prayer) get() Prayer {
	table := "ActivePrayers"

	resp := getItem(p.PhoneNumber, table)

	err := attributevalue.UnmarshalMap(resp.Item, &p)
	if err != nil {
		log.Fatalf("unmarshal failed, %v", err)
	}

	return p
}

func (per Person) delete() {
	tables := []string{"Members", "Intercessors"}

	for _, table := range tables {
		delItem(per.PhoneNumber, table)
	}
}

func (p Prayer) delete() {
	table := "ActivePrayers"

	delItem(p.PhoneNumber, table)
}

func (per Person) put(table string) {
	data := map[string]types.AttributeValue{
		"Name":        &types.AttributeValueMemberS{Value: per.Name},
		"PhoneNumber": &types.AttributeValueMemberS{Value: per.PhoneNumber},
		"PrayerLimit": &types.AttributeValueMemberS{Value: per.PrayerLimit},
		"SetupStage":  &types.AttributeValueMemberS{Value: per.SetupStage},
		"SetupStatus": &types.AttributeValueMemberS{Value: per.SetupStatus},
	}

	putItem(table, data)
}

func (p Prayer) put() {
	table := "ActivePrayers"

	data := map[string]types.AttributeValue{
		"People":      &types.AttributeValueMemberSS{Value: p.People},
		"PhoneNumber": &types.AttributeValueMemberS{Value: p.PhoneNumber},
		"Request":     &types.AttributeValueMemberS{Value: p.Request},
	}

	putItem(table, data)
}

func getItem(phone, table string) *dynamodb.GetItemOutput {
	client := getDdbClient()

	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			"PhoneNumber": &types.AttributeValueMemberS{Value: phone},
		},
	})

	if err != nil {
		log.Fatalf("unable to get item: %v", err)
	}

	return resp
}

func putItem(table string, data map[string]types.AttributeValue) {
	client := getDdbClient()

	_, err := client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &table,
		Item:      data,
	})
	if err != nil {
		log.Fatalf("unable to put item: %v", err)
	}
}

func delItem(phone, table string) {
	client := getDdbClient()

	_, err := client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			"PhoneNumber": &types.AttributeValueMemberS{Value: phone},
		},
	})

	if err != nil {
		log.Fatalf("unable to delete item: %v", err)
	}
}

func signUp(txt TextMessage, per Person) {
	const (
		nameRequest               = "Send your name or 2 to stay anonymous"
		memberType                = "Send 1 for prayer request or 2 to be added to the intercessors list (to pray for others)"
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

func delUser(per Person) {
	per.delete()
	per.sendMessage("You have been removed from prayer texter. If you ever want to sign back up, text the word pray to this number.")
}

func prayerRequest(txt TextMessage) {
	// p1 := Person{
	// 	Name:        "Person 1",
	// 	PhoneNumber: "111-111-1111",
	// }
	// p2 := Person{
	// 	Name:        "Person 2",
	// 	PhoneNumber: "222-222-2222",
	// }
	// p3 := Person{
	// 	Name:        "Person 3",
	// 	PhoneNumber: "222-222-2222",
	// }

	pryr := Prayer{
		People:      []string{"111-111-1111", "222-222-2222", "333-333-3333"},
		PhoneNumber: "888-888-8888",
		Request:     txt.Body,
	}

	pryr.put()

	for _, p := range pryr.People {
		sendText(pryr.Request, p)
	}
}

func mainFlow(txt TextMessage) error {
	// if text body != pray or stop or cancel && phone number in members: start new prayer request process
	// else: drop text???
	per := Person{
		PhoneNumber: txt.PhoneNumber,
	}

	per = per.get("Members")

	if strings.ToLower(txt.Body) == "pray" || per.SetupStatus == "in-progress" {
		signUp(txt, per)
	} else if strings.ToLower(txt.Body) == "cancel" || strings.ToLower(txt.Body) == "stop" {
		delUser(per)
	} else if per.SetupStatus == "completed" {
		prayerRequest(txt)
	} else if per.SetupStatus == "" {
		log.Printf("%v is not a registered user, dropping message", per.PhoneNumber)
	}

	return nil
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (
	events.APIGatewayProxyResponse, error) {
	txt := TextMessage{}

	err := json.Unmarshal([]byte(req.Body), &txt)
	if err != nil {
		log.Fatalf("failed to unmarshal api gateway request. error - %s\n", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	err = mainFlow(txt)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Prayer flow completed successfully",
	}, nil
}

func main() {
	lambda.Start(handler)
}
