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

const (
	InitialQuestion = "Send 1 for prayer request or 2 to be added to the intercessors list (to " +
		"pray for others)"
	PrayerRequestInstructions = "You are now signed up to send prayer requests! Please send them " +
		"directly to this number."
	NameRequest      = "Send your name or 2 to stay anonymous"
	PrayerNumRequest = "Send the max number of prayer texts you are willing to receive and pray " +
		"for per week."
	IntercessorInstructions = "You are now signed up to receive prayer requests. Please try to " +
		"pray for the requests ASAP. Once you are done praying, send 'prayed' back to this number" +
		"for confirmation."
	PrayerIntro        = "Hello! Please pray for this person:"
	PrayerConfirmation = "Your prayer request has been sent out!"
)

type TextMessage struct {
	Body        string `json:"body"`
	PhoneNumber string `json:"phone-number"`
}

type Person struct {
	Name        string
	PhoneNumber string
	PrayerCount string
	SetupStage  string
}

type Prayer struct {
	People      []string
	PhoneNumber string
	Request     string
}

/// NOTES
/// created get and delete methods for Person and Prayer
/// need to create put methods for the same
/// need to decide whether to keep this way as get methods for the 2 are identical!
/// maybe they don't need to be methods??? but then I can't use interfaces

func (p Person) SendPrayer(prayer Prayer) {
	body := PrayerIntro + "\n" + prayer.Request
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

func (p Person) get(table string) (Person, error) {
	// handle error logging better; if both functions fail only 2nd error is logged
	resp, err := getItem(p.PhoneNumber, table)
	if err != nil {
		log.Fatalf("unable to get %v from %v", p.PhoneNumber, table)
	}

	err = attributevalue.UnmarshalMap(resp.Item, &p)
	if err != nil {
		log.Fatalf("unmarshal failed, %v", err)
	}

	return p, err
}

func (p Prayer) get(table string) (Prayer, error) {
	// handle error logging better; if both functions fail only 2nd error is logged
	resp, err := getItem(p.PhoneNumber, table)
	if err != nil {
		log.Fatalf("unable to get %v from %v", p.PhoneNumber, table)
	}

	err = attributevalue.UnmarshalMap(resp.Item, &p)
	if err != nil {
		log.Fatalf("unmarshal failed, %v", err)
	}

	return p, err
}

func (p Person) delete() error {
	// handle error logging better; if both deletes fail only 2nd error is logged
	tables := []string{"Members", "Intercessors"}

	var err error

	for _, table := range tables {
		err = delItem(p.PhoneNumber, table)
		if err != nil {
			log.Fatalf("unable to delete %v from %v table", p.PhoneNumber, table)
		}
	}

	return err
}

func (p Prayer) delete() error {
	table := "ActivePrayers"

	err := delItem(p.PhoneNumber, table)
	if err != nil {
		log.Fatalf("unable to delete %v from %v", p.PhoneNumber, table)
	}

	return err
}

// func getPrayer(phone string) (Prayer, error) {
// 	client := getDdbClient()

// 	table := "ActivePrayers"

// 	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
// 		TableName: &table,
// 		Key: map[string]types.AttributeValue{
// 			"PhoneNumber": &types.AttributeValueMemberS{Value: phone},
// 		},
// 	})
// 	if err != nil {
// 		log.Fatalf("unable to get item: %v", err)
// 		return Prayer{}, err
// 	}

// 	var prayer Prayer

// 	err = attributevalue.UnmarshalMap(resp.Item, &prayer)
// 	if err != nil {
// 		log.Fatalf("unmarshal failed, %v", err)
// 		return Prayer{}, err
// 	}

// 	return prayer, err
// }

func putPrayer(p Prayer) error {
	/// handle case where multiple active prayers get sent in by same phone number
	client := getDdbClient()

	table := "ActivePrayers"

	_, err := client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &table,
		Item: map[string]types.AttributeValue{
			"PhoneNumber": &types.AttributeValueMemberS{Value: p.PhoneNumber},
			"Request":     &types.AttributeValueMemberS{Value: p.Request},
			"People":      &types.AttributeValueMemberSS{Value: p.People},
		},
	})
	if err != nil {
		log.Fatalf("unable to put item: %v", err)
	}

	return err
}

// func getPerson(phone string, table string) (Person, error) {
// 	client := getDdbClient()

// 	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
// 		TableName: &table,
// 		Key: map[string]types.AttributeValue{
// 			"PhoneNumber": &types.AttributeValueMemberS{Value: phone},
// 		},
// 	})
// 	if err != nil {
// 		log.Fatalf("unable to get item: %v", err)
// 		return Person{}, err
// 	}

// 	var person Person

// 	err = attributevalue.UnmarshalMap(resp.Item, &person)
// 	if err != nil {
// 		log.Fatalf("unmarshal failed, %v", err)
// 		return Person{}, err
// 	}

// 	return person, err
// }

func putPerson(p Person, table string) error {
	client := getDdbClient()

	_, err := client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &table,
		Item: map[string]types.AttributeValue{
			"Name":        &types.AttributeValueMemberS{Value: p.Name},
			"PhoneNumber": &types.AttributeValueMemberS{Value: p.PhoneNumber},
			"PrayerCount": &types.AttributeValueMemberN{Value: p.PrayerCount},
			"SetupStage":  &types.AttributeValueMemberN{Value: p.SetupStage},
		},
	})
	if err != nil {
		log.Fatalf("unable to put item: %v", err)
	}

	return err
}

func getItem(phone, table string) (*dynamodb.GetItemOutput, error) {
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

	return resp, err
}

func putItem(phone, table string, data map[string]types.AttributeValue) error {
	client := getDdbClient()

	_, err := client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &table,
		Item:      data,
	})
	if err != nil {
		log.Fatalf("unable to put item: %v", err)
	}

	return err
}

func delItem(phone, table string) error {
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

	return err
}

func SignUp(txt TextMessage) {
	if txt.Body == "pray" {
		new := Person{
			Name:        "anonomysous",
			PhoneNumber: txt.PhoneNumber,
			PrayerCount: "0",
			SetupStage:  "1",
		}

	}
}

func mainFlow(txt TextMessage) error {

	// if text body == pray: start NEW sign up process (overwrite any existing sign up process)
	// if text body == stop or cancel: remove from members, intercessors, and sign ups
	// if text body != pray or stop or cancel && phone number in active sign ups: continue sign up flow
	// if text body != pray or stop or cancel && phone number in members: start new prayer request process
	// else: drop text???

	return err
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (
	events.APIGatewayProxyResponse, error) {
	txt := TextMessage{}

	err := json.Unmarshal([]byte(req.Body), &txt)
	if err != nil {
		log.Fatalf("failed to unmarshal api gateway request. error - %s\n", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	err = MainFlow(txt)
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
