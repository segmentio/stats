package grpcstats

import (
	"context"
	"log"

	"google.golang.org/grpc"
)

func MetricsServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// // Get the metadata from the incoming context
		// md, ok := metadata.FromIncomingContext(ctx)
		// if !ok {
		// 	return nil, fmt.Errorf("couldn't parse incoming context metadata")
		// }

		// m := &metrics{}

		log.Printf("-->> metrics interceptor hit")
		log.Print(ctx)
		log.Print(req)
		log.Print(info)

		h, err := handler(ctx, req)

		return h, err
	}
}

func WithMetricsInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(MetricsServerInterceptor())
}
