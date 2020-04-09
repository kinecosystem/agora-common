package util

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kinecosystem/agora-common/aws/dynamodb/test"
)

func TestErrors_NoMatch(t *testing.T) {
	assert.False(t, IsConditionalCheckFailed(nil))
	assert.False(t, IsConditionalCheckFailed(errors.New("blah")))

	assert.Nil(t, MapConditionalCheckFailed(nil, errors.New("blah")))

	fooErr := errors.New("foo")
	assert.Equal(t, fooErr, MapConditionalCheckFailed(fooErr, errors.New("bar")))
}

func TestErrors_Match(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	db, cleanupFunc, err := test.StartDynamoDB(pool)
	require.NoError(t, err)
	defer cleanupFunc()

	_, err = db.CreateTableRequest(&dynamodb.CreateTableInput{
		TableName: aws.String("test-table"),
		KeySchema: []dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("k"),
				KeyType:       dynamodb.KeyTypeHash,
			},
		},
		AttributeDefinitions: []dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("k"),
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
			"k": {
				S: aws.String("hello"),
			},
			"v": {
				S: aws.String("world"),
			},
		},
	}).Send(context.Background())
	require.NoError(t, err)

	_, err = db.PutItemRequest(&dynamodb.PutItemInput{
		TableName: aws.String("test-table"),
		Item: map[string]dynamodb.AttributeValue{
			"k": {
				S: aws.String("hello"),
			},
			"v": {
				S: aws.String("world"),
			},
		},
		ConditionExpression: aws.String("attribute_not_exists(k)"),
	}).Send(context.Background())
	assert.NotNil(t, err)
	fmt.Println(err)
	assert.True(t, IsConditionalCheckFailed(err))

	notFound := errors.New("not found")
	assert.Equal(t, notFound, MapConditionalCheckFailed(err, notFound))
}
