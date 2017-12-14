package gotravelservice

import (
	"context"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/gotravelservice/trip"
	"github.com/go-kit/kit/log"
)

// Middleware is a service middleware, similar to endpoint middleware
type Middleware func(Service) Service

// NewLoggingMiddleware given a logger returns a service middleware
// that logs service methods calls
func NewLoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) TripPlan(ctx context.Context, tc trip.Configuration) (t trip.Trip, err error) {
	defer func() {
		mw.logger.Log(
			"method", "TripPlan",
			"apiKey", tc.APIKey,
			"schedule", t.Schedule,
			"err", err,
		)
	}()
	return mw.next.TripPlan(ctx, tc)
}
