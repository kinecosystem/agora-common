package app

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	s3test "github.com/kinecosystem/agora-common/aws/s3/test"
)

func TestLocalLoader(t *testing.T) {
	f, err := ioutil.TempFile("", "file_loader")
	require.NoError(t, err)

	_, err = f.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	fmt.Println(f.Name())
	defer os.Remove(f.Name())

	var loader LocalLoader

	for _, u := range []string{
		f.Name(),                           // /tmp/file
		fmt.Sprintf("file://%s", f.Name()), // file:///temp/file
		fmt.Sprintf("file://%s", path.Join("localhost/", f.Name())), // file://localhost/temp/file
	} {
		contents, err := loader.Load(getURL(t, u))
		assert.NoError(t, err, "failed to load %s", u)
		assert.Equal(t, []byte("hello"), contents)
	}
}

func TestS3Loader(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	client, cleanupFunc, err := s3test.StartS3(pool)
	require.NoError(t, err)
	defer cleanupFunc()

	_, err = client.CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: aws.String("bucket"),
	}).Send(context.Background())
	require.NoError(t, err)

	l := S3Loader{
		s3: client,
	}

	_, err = l.Load(getURL(t, "s3://bucket/configs/example"))
	assert.NotNil(t, err)

	_, err = client.PutObjectRequest(&s3.PutObjectInput{
		Body:   bytes.NewReader([]byte("hello")),
		Bucket: aws.String("bucket"),
		Key:    aws.String("configs/example"),
	}).Send(context.Background())
	require.NoError(t, err)

	contents, err := l.Load(getURL(t, "s3://bucket/configs/example"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), contents)
}

func TestS3Loader_BadURL(t *testing.T) {
	l := S3Loader{}

	for _, u := range []string{
		"file:///file",
		"bucket/ket",
		"s3://bucket",
		"s3:///my/key",
	} {
		_, err := l.Load(getURL(t, u))
		assert.NotNil(t, err, "expected url to fail: %s", u)
	}
}

func getURL(t *testing.T, u string) *url.URL {
	url, err := url.Parse(u)
	require.NoError(t, err)

	return url
}
