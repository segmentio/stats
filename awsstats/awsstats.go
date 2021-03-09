package awsstats

import (
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/segmentio/go-snakecase"
	"github.com/segmentio/stats/v4"
)

func InjectHandlers(engine *stats.Engine, sess *session.Session) {
	sess.Handlers.Complete.PushBack(func(request *request.Request) {
		now := time.Now()
		tags := requestTags(request)

		engine.Incr("aws.api.complete", tags...)
		engine.Observe("aws.api.latency", now.Sub(request.Time), tags...)
	})

	sess.Handlers.CompleteAttempt.PushBack(func(request *request.Request) {
		now := time.Now()
		tags := requestTags(request)

		engine.Incr("aws.api.attempt.complete", tags...)
		engine.Observe("aws.api.attempt.latency", now.Sub(request.AttemptTime), tags...)
	})
}

func requestTags(req *request.Request) []stats.Tag {
	tags := []stats.Tag{
		stats.T("region", aws.StringValue(req.Config.Region)),
		stats.T("service", snakecase.Snakecase(req.ClientInfo.ServiceID)),
		stats.T("api_version", req.ClientInfo.APIVersion),
		stats.T("has_error", strconv.FormatBool(req.Error != nil)),
		stats.T("max_retries_exceeded", strconv.FormatBool(req.RetryCount >= req.MaxRetries())),
	}

	if req.Operation != nil {
		tags = append(tags, stats.T("method", snakecase.Snakecase(req.Operation.Name)))
	}

	if req.Error != nil {
		var awsErr awserr.Error
		if errors.As(req.Error, &awsErr) {
			tags = append(tags, stats.T("error_code", snakecase.Snakecase(awsErr.Code())))
		}
	}

	if req.HTTPResponse != nil {
		tags = append(tags, stats.T("http_response_status_code", strconv.Itoa(req.HTTPResponse.StatusCode)))
	}

	return tags
}
