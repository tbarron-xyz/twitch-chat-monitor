package main

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	mgo "gopkg.in/mgo.v2"
)

const DynamoDbTableName = "emotehistory"
const DynamoDbTimeToLiveInSeconds = 86400 // 1 day

type DatabaseWrapper interface {
	Insert(...interface{}) error
}

type DynamoForWrapping struct {
	dynamodb *dynamodb.DynamoDB
}

func (d *DynamoForWrapping) Insert(args ...interface{}) error {
	var snap, ok = args[0].(snapshot)
	if !ok {
		fmt.Println("Insert called but argument not a snapshot")
	}
	dataAsMapStringAttributeValue := map[string]*dynamodb.AttributeValue{}
	for emote, count := range snap.Data {
		dataAsMapStringAttributeValue[emote] = &dynamodb.AttributeValue{N: aws.String(strconv.Itoa(count))}
	}
	expireTime := strconv.Itoa(int(snap.Time) + DynamoDbTimeToLiveInSeconds)
	var item = map[string]*dynamodb.AttributeValue{
		"Time": &dynamodb.AttributeValue{N: aws.String(expireTime)},
		"Data": &dynamodb.AttributeValue{M: dataAsMapStringAttributeValue},
	}
	putItemInput := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(DynamoDbTableName),
	}
	result, err := d.dynamodb.PutItem(putItemInput)
	if err != nil {
		return err
	}
	_ = result
	return nil
}

func (d *DynamoForWrapping) ScanTable() []snapshot {
	params := &dynamodb.ScanInput{
		TableName: aws.String(DynamoDbTableName),
	}
	scanResult, err := d.dynamodb.Scan(params)
	if err != nil {
		fmt.Errorf("failed to make Query API call, %v", err)
	}

	obj := []snapshot{}
	err = dynamodbattribute.UnmarshalListOfMaps(scanResult.Items, &obj)
	if err != nil {
		EH(err)
	}
	return obj
}

func dynamodbDatabase() DatabaseWrapper {
	// Create a DynamoDB client from just a session.
	mySession, err := session.NewSession()
	if err != nil {
		EH(err)
	}
	svc1 := dynamodb.New(mySession, &aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
	})

	wrapped := &DynamoForWrapping{dynamodb: svc1}

	return wrapped
}

func mongoDatabase() DatabaseWrapper {
	var mgoclient, err = mgo.Dial("mongodb://localhost/kappa")
	if err != nil {
		panic("failed to connect to mongo")
	}
	var mgodb = mgoclient.DB("")
	var snapsCollection = mgodb.C("Snapshots")
	return snapsCollection
}

type snapshot struct {
	Time int64          `redis:"time"`
	Data map[string]int `redis:"data"`
}
