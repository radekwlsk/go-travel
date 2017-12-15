package gotravelendpoint

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
)

// NewLoggingMiddleware returns endpoint middleware that logs
// information about duration of each call and error if any occurred
func NewLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				logger.Log(
					"err", err,
					"took", time.Since(begin),
				)
			}(time.Now())

			return next(ctx, request)
		}
	}
}
