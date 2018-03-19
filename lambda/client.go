package lambda

import (
	"bufio"
	"os"
	"time"

	"github.com/segmentio/stats"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) HandleMeasures(t time.Time, measures ...stats.Measure) {
	f := bufio.NewWriter(os.Stdout)
	defer f.Flush()

	for _, measure := range measures {
		var b []byte
		f.Write(AppendMeasure(b, t, measure))
		f.Flush()
	}
}
