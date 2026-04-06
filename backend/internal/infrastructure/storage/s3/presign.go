package s3

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Presigner struct {
	bucket string
	client *s3.Client
}

func NewPresigner(bucket string, client *s3.Client) *Presigner {
	return &Presigner{bucket: bucket, client: client}
}

func (p *Presigner) PresignGetObject(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	ps := s3.NewPresignClient(p.client)
	out, err := ps.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &p.bucket,
		Key:    &key,
	}, func(o *s3.PresignOptions) {
		o.Expires = expiresIn
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}
