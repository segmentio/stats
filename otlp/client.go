package otlp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/protobuf/proto"
)

type Client interface {
	Handle(context.Context, *colmetricpb.ExportMetricsServiceRequest) error
}

// HTTPClient implements the Client interface and is used to export metrics to
// an OpenTelemetry Collector through the HTTP interface.
//
// The current implementation is a fire and forget approach where we do not retry
// or buffer any failed-to-flush data on the client.
type HTTPClient struct {
	client   *http.Client
	endpoint string
}

func NewHTTPClient(endpoint string) *HTTPClient {
	return &HTTPClient{
		// TODO: add sane default timeout configuration.
		client:   http.DefaultClient,
		endpoint: endpoint,
	}
}

func (c *HTTPClient) Handle(ctx context.Context, request *colmetricpb.ExportMetricsServiceRequest) error {
	rawReq, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %s", err)
	}

	httpReq, err := newRequest(ctx, c.endpoint, rawReq)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %s", err)
	}

	return c.do(httpReq)
}

// TODO: deal with requests failures and retries. We potentially want to implement
//
//	some kind of retry mechanism with expotential backoff + short time window.
func (c *HTTPClient) do(req *http.Request) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	msg, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send data to collector, code: %d, error: %s",
			resp.StatusCode,
			string(msg),
		)
	}

	return nil
}

func newRequest(ctx context.Context, endpoint string, data []byte) (*http.Request, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("User-Agent", "segmentio/stats")

	req.Body = io.NopCloser(bytes.NewReader(data))
	return req, nil
}
