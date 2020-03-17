package test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3(t *testing.T) {
	bucket := "bucket"
	key := "file"
	contents := []byte("data")

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	client, cleanupFunc, err := StartS3(pool)
	require.NoError(t, err)
	defer cleanupFunc()

	_, err = client.CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: aws.String("bucket"),
	}).Send(context.Background())
	require.NoError(t, err)

	listBucketsResp, err := client.ListBucketsRequest(&s3.ListBucketsInput{}).Send(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, len(listBucketsResp.Buckets))
	assert.Equal(t, bucket, *listBucketsResp.Buckets[0].Name)

	_, err = client.PutObjectRequest(&s3.PutObjectInput{
		ACL:           s3.ObjectCannedACLPrivate,
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(contents),
		ContentLength: aws.Int64(int64(len(contents))),
	}).Send(context.Background())
	require.NoError(t, err)

	listObjectsResp, err := client.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
	}).Send(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, len(listObjectsResp.Contents))
	assert.Equal(t, key, *listObjectsResp.Contents[0].Key)

	getResp, err := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}).Send(context.Background())
	require.NoError(t, err)

	actualContents, err := ioutil.ReadAll(getResp.Body)
	require.NoError(t, err)
	assert.Equal(t, contents, actualContents)

	_, err = client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}).Send(context.Background())
	require.NoError(t, err)

	listObjectsResp, err = client.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
	}).Send(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, len(listObjectsResp.Contents))

	_, err = client.DeleteBucketRequest(&s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}).Send(context.Background())
	require.NoError(t, err)
}
