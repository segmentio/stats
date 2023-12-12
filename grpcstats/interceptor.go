package grpcstats

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/stats/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func NewInterceptor() grpc.UnaryServerInterceptor {
	return NewInterceptorWith(stats.DefaultEngine)
}

func NewInterceptorWith(eng *stats.Engine) grpc.UnaryServerInterceptor {
	// return &interceptor{
	// 	eng: eng,
	// }

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		log.Printf("-->> metrics interceptor hit")
		log.Print(ctx)
		log.Print(req)
		log.Print(info)

		// Get the metadata from the incoming context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, fmt.Errorf("couldn't parse incoming context metadata")
		}

		for k, v := range md {
			fmt.Printf("md.%v: %v\n", k, v)
		}

		fmt.Println("info.FullMethod")
		fmt.Println(info.FullMethod)

		// 1. Create a metrics obj
		m := &metrics{}

		// 2. Observe response
		// start := time.Now()
		w := &responseWriter{
			eng:     eng,
			req:     req,
			info:    info,
			metrics: m,
			start:   time.Now(),
		}
		defer w.complete()

		// 3. Observe Request
		b := &requestBody{
			eng:     eng,
			req:     req,
			info:    info,
			metrics: m,
		}
		defer b.close()

		// 4. Continue middleware call chain
		h, err := handler(ctx, req)

		return h, err
	}
}

type responseWriter struct {
	start       time.Time
	eng         *stats.Engine
	req         interface{}
	info        *grpc.UnaryServerInfo
	metrics     *metrics
	status      int
	bytes       int
	wroteHeader bool
	wroteStats  bool
}

func (w *responseWriter) complete() {
	if w.wroteStats {
		return
	}
	w.wroteStats = true

	// now := time.Now()
	// w.metrics.observeResponse(res, "write", w.bytes, now.Sub(w.start))
	w.eng.ReportAt(w.start, w.metrics)
}

// func metricsServerInterceptor() grpc.UnaryServerInterceptor {
// 	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

// 		log.Printf("-->> metrics interceptor hit")
// 		log.Print(ctx)
// 		log.Print(req)
// 		log.Print(info)

// 		// Get the metadata from the incoming context
// 		md, ok := metadata.FromIncomingContext(ctx)
// 		if !ok {
// 			return nil, fmt.Errorf("couldn't parse incoming context metadata")
// 		}

// 		for k, v := range md {
// 			fmt.Printf("md.%v: %v\n", k, v)
// 		}

// 		m := &metrics{}

// 		start := time.Now()
// 		w := &responseWriter{
// 			eng:     h.eng,
// 			req:     req,
// 			info:    info,
// 			metrics: m,
// 			start:   time.Now(),
// 		}
// 		defer w.complete()

// 		h, err := handler(ctx, req)

// 		return h, err
// 	}
// }

type requestBody struct {
	eng     *stats.Engine
	req     interface{}
	info    *grpc.UnaryServerInfo
	metrics *metrics
	once    sync.Once
}

func (r *requestBody) close() {
	r.once.Do(r.complete)
}

func (r *requestBody) complete() {
	r.metrics.observeRequest(r.info)
}

// func WithMetricsInterceptor() grpc.ServerOption {
// 	return grpc.UnaryInterceptor(MetricsServerInterceptor())
// }
