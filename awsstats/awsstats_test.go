package awsstats

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/segmentio/ksuid"
	"github.com/segmentio/stats/v4"
	"github.com/segmentio/stats/v4/statstest"
	"github.com/stretchr/testify/require"
)

func TestInjectHandlers(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	sess := newLocalstackSession(t)
	InjectHandlers(e, sess)

	s3api := s3.New(sess, aws.NewConfig().WithS3ForcePathStyle(true))

	id := ksuid.New()
	input := s3.CreateBucketInput{Bucket: aws.String(id.String())}
	s3api.CreateBucket(&input)
	s3api.CreateBucket(&input) // should fail with "BucketAlreadyExists" error code

	e.Flush()

	if len(h.Measures()) == 0 {
		t.Fatal("no measures recorded")
	}

	for _, measure := range h.Measures() {
		for _, field := range measure.Fields {
			t.Logf("%s.%s = %v", measure.Name, field.Name, field.Value)
			for _, tag := range measure.Tags {
				t.Log(">", tag.String())
			}
		}
	}
}

func newLocalstackSession(t *testing.T) *session.Session {
	conf := aws.NewConfig().
		WithEndpoint("http://localhost:4566").
		WithRegion(endpoints.UsWest2RegionID).
		WithCredentials(credentials.NewStaticCredentials("a", "b", "c"))

	sess, err := session.NewSessionWithOptions(session.Options{Config: *conf})
	require.NoError(t, err)
	return sess
}
