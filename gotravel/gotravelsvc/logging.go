package gotravelsvc

import (
	"time"

	"context"

	"github.com/go-kit/kit/log"
	"github.com/kr/pretty"
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

func (mw loggingMiddleware) TripPlan(ctx context.Context, tc TripConfiguration) (trip Trip, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "TripPlan",
			"input", pretty.Sprint(tc),
			"output", pretty.Sprint(trip),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	trip, err = mw.next.TripPlan(ctx, tc)
	return
}
