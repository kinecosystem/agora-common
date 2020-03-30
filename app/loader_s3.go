package app

import (
	"context"
	"io/ioutil"
	"net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/pkg/errors"
)

// S3Loader is a FileLoader that loads files from S3.
type S3Loader struct {
	s3 s3iface.ClientAPI
}

// Load implements FileLoader.Load.
func (l S3Loader) Load(url *url.URL) ([]byte, error) {
	if url.Scheme != "s3" {
		return nil, errors.Errorf("invalid scheme: %s", url.Scheme)
	}
	if url.Host == "" {
		return nil, errors.New("missing bucket")
	}
	if url.Path == "" {
		return nil, errors.New("missing key")
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFunc()

	resp, err := l.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(url.Host),
		Key:    aws.String(url.Path[1:]), // uh you suck
	}).Send(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load %s", url.String())
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func init() {
	var init sync.Once

	var loader FileLoader
	var initErr error

	ctr := func() (FileLoader, error) {
		init.Do(func() {
			cfg, err := external.LoadDefaultAWSConfig()
			if err != nil {
				initErr = errors.Wrap(err, "failed to initialize S3Loader")
				return
			}

			loader = &S3Loader{s3: s3.New(cfg)}
		})

		if initErr != nil {
			return nil, initErr
		}

		return loader, nil
	}

	RegisterFileLoader("s3", ctr)
}
