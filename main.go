package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

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

var version string // do not remove or modify

const (
	NameRequest = "Send your name or 2 to stay anonymous"
	InitialQuestion = "Send 1 for prayer request or 2 to be added to the intercessors list (to pray for others)"
	PrayerRequestInstructions = "You are now signed up to send prayer requests! Please send them directly to this number."
	PrayerNumRequest = "Send the max number of prayer texts you are willing to receive and pray for per week."
	IntercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the requests ASAP. Once you are done praying, send 'prayed' back to this number for confirmation."
	PrayerIntro = "Hello! Please pray for this person:"
	PrayerConfirmation = "Your prayer request has been sent out!"
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
}

type Prayer struct {
	People      []string
	PhoneNumber string
	Request     string
}

func (p Person) sendMessage(body string) {
	sendText(body, p.PhoneNumber)
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

func (p Person) get(table string) Person {
	resp := getItem(p.PhoneNumber, table)

	err := attributevalue.UnmarshalMap(resp.Item, &p)
	if err != nil {
		log.Fatalf("unmarshal failed, %v", err)
	}

	return p
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

func (p Person) delete() {
	tables := []string{"Members", "Intercessors"}

	for _, table := range tables {
		delItem(p.PhoneNumber, table)
	}
}

func (p Prayer) delete() {
	table := "ActivePrayers"

	delItem(p.PhoneNumber, table)
}

func (p Person) put(table string) {
	data := map[string]types.AttributeValue{
		"Name":        &types.AttributeValueMemberS{Value: p.Name},
		"PhoneNumber": &types.AttributeValueMemberS{Value: p.PhoneNumber},
		"PrayerLimit": &types.AttributeValueMemberN{Value: p.PrayerLimit},
		"SetupStage":  &types.AttributeValueMemberN{Value: p.SetupStage},
	}

	putItem(table, data)
}

func (p Prayer) put() {
	table := "ActivePrayers"

	data := map[string]types.AttributeValue{
		"PhoneNumber": &types.AttributeValueMemberS{Value: p.PhoneNumber},
		"Request":     &types.AttributeValueMemberS{Value: p.Request},
		"People":      &types.AttributeValueMemberSS{Value: p.People},
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

func signUp(txt TextMessage, p Person) {
	switch {
	// stage 1
	case txt.Body == "pray" && p.SetupStage == "":
		p.SetupStage = "1"
		p.put("Members")
		p.sendMessage(NameRequest)
	// stage 2 name request
	case txt.Body != "2" && p.SetupStage == "1":
		p.SetupStage = "2"
		p.Name = txt.Body
		p.put("Members")
		p.sendMessage(InitialQuestion)
	// stage 2 name request
	case txt.Body == "2" && p.SetupStage == "1":
		p.SetupStage = "2"
		p.Name = "anonymous"
		p.put("Members")
		p.sendMessage(InitialQuestion)
	// final stage for member sign up
	case txt.Body == "1" && p.SetupStage == "2":
		p.SetupStage = "99"
		p.put("Members")
		p.sendMessage(PrayerRequestInstructions)
	// stage 3 intercessor sign up
	case txt.Body == "2" && p.SetupStage == "2":
		p.SetupStage = "3"
		p.put("Members")
		p.put("Intercessors")
		p.sendMessage(PrayerNumRequest)
	// final stage for intercessor sign up
	case p.SetupStage == "3":
		///
	}
}

func mainFlow(txt TextMessage) error {

	// if text body == pray: start NEW sign up process (overwrite any existing sign up process)
	// if text body == stop or cancel: remove from members, intercessors, and sign ups
	// if text body != pray or stop or cancel && phone number in active sign ups: continue sign up flow
	// if text body != pray or stop or cancel && phone number in members: start new prayer request process
	// else: drop text???

	// person := Person{
	// 	Name: "Matt",
	// 	PhoneNumber: "657-217-1678",
	// 	PrayerLimit: "7",
	// 	SetupStage: "4",
	// }

	// prayer := Prayer{
	// 	People: []string{"person1", "person2", "person3"},
	// 	PhoneNumber: "777-777-7777",
	// 	Request: "Please help me!!!",
	// }

	// person.put("Members")
	// person.put("Intercessors")
	// prayer.put()

	// var newperson Person
	// newperson.PhoneNumber = "657-217-1678"
	// newperson = newperson.get("Members")

	// fmt.Printf("newperson: %v\n", newperson)

	// person.delete()
	// prayer.delete()

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
