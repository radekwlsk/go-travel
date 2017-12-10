package gotravelsvc

import (
	"context"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/trip"
	"github.com/go-kit/kit/log"
)

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

// NewLoggingMiddleware returns Service middleware that logs
// information about each method execution including:
// method name, input, output, error if present and time of execution
func NewLoggingMiddleware(s Service, logger log.Logger) Service {
	return loggingMiddleware{logger, s}
}

func (mw loggingMiddleware) TripPlan(ctx context.Context, tc trip.Configuration) (trip trip.Trip, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "TripPlan",
			"apiKey", tc.APIKey,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	trip, err = mw.next.TripPlan(ctx, tc)
	return
}
