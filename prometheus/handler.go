package prometheus

import (
	"net/http"
	"sync"

	"github.com/segmentio/stats"
)

type Handler struct {
	mutex   sync.RWMutex
	metrics map[string]*metric
}

func (h *Handler) HandleMetric(m *stats.Metric) {

}

func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

}
