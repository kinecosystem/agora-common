package test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/require"
)

func TestStartDynamoDB(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	db, cleanupFunc, err := StartDynamoDB(pool)
	require.NoError(t, err)
	defer cleanupFunc()

	_, err = db.CreateTableRequest(&dynamodb.CreateTableInput{
		TableName: aws.String("test-table"),
		KeySchema: []dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("key"),
				KeyType:       dynamodb.KeyTypeHash,
			},
		},
		AttributeDefinitions: []dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("key"),
				AttributeType: dynamodb.ScalarAttributeTypeS,
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}).Send(context.Background())
	require.NoError(t, err)

	_, err = db.PutItemRequest(&dynamodb.PutItemInput{
		TableName: aws.String("test-table"),
		Item: map[string]dynamodb.AttributeValue{
			"key": {
				S: aws.String("hello"),
			},
			"val": {
				S: aws.String("world"),
			},
		},
	}).Send(context.Background())
	require.NoError(t, err)

	resp, err := db.GetItemRequest(&dynamodb.GetItemInput{
		TableName: aws.String("test-table"),
		Key: map[string]dynamodb.AttributeValue{
			"key": {S: aws.String("hello")},
		},
	}).Send(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp.Item["val"])
	require.NotNil(t, resp.Item["val"].S)
	require.Equal(t, "world", *resp.Item["val"].S)
}
