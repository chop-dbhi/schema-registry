package transport

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"golang.org/x/net/context"
)

// Generic logger middleware.
func endpointLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			start := time.Now()
			rep, err := next(ctx, request)
			logger.Log("elapsed", uint64(time.Since(start)/time.Microsecond), "error", err)
			return rep, err
		}
	}
}
