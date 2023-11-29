package grpcstats

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func MetricsServerInterceptor() grpc.UnaryServerInterceptor {
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

		// m := &metrics{}

		h, err := handler(ctx, req)

		return h, err
	}
}

func WithMetricsInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(MetricsServerInterceptor())
}
