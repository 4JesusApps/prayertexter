package main

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type IntercessorList struct {
	General         string
	IntercessorList []Person
}

const (
	intercessorListAttribute = "Name"
	intercessorListKey       = "IntercessorsList"
	intercessorListTable     = "General"
)

func getIntercessors() []Person {
	out := getAllItems(intercessorsTable)

	intercessors := []Person{}

	err := attributevalue.UnmarshalListOfMaps(out.Items, &intercessors)
	if err != nil {
		log.Fatalf("unmarshal failed for scan items on intercessors table, %v", err)
	}

	return intercessors
}

func (i IntercessorList) get() IntercessorList {
	resp := getItem(intercessorListAttribute, intercessorListKey, intercessorListTable)

	err := attributevalue.UnmarshalMap(resp.Item, &i)
	if err != nil {
		log.Fatalf("unmarshal failed for get person, %v", err)
	}

	return i
}

func (i IntercessorList) put() {
	data, err := attributevalue.MarshalMap(i)
	if err != nil {
		log.Fatalf("unmarshal failed for put IntercessorsList, %v", err)
	}

	putItem(intercessorListTable, data)
}
