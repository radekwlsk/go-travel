package types

import (
	"math"
	"time"

	traveltypes "github.com/afrometal/go-travel/gotravel/gotravelsvc/types"
)

type Result struct {
	path       traveltypes.Path
	time       time.Duration
	distance   int64
	priorities int
	visitTimes VisitTimes
}

func NewResult(path traveltypes.Path, dur time.Duration, dist int64, prio int, times VisitTimes) Result {
	return Result{
		path:       path,
		time:       dur,
		distance:   dist,
		priorities: prio,
		visitTimes: times,
	}
}

func NewEmptyResult() Result {
	return Result{
		path:       traveltypes.NewDummyPath(),
		time:       time.Duration(math.MaxInt64),
		distance:   math.MaxInt64,
		priorities: 0,
		visitTimes: VisitTimes{},
	}
}

func (r *Result) Path() traveltypes.Path {
	return r.path
}

func (r *Result) Time() time.Duration {
	return r.time
}

func (r *Result) Distance() int64 {
	return r.distance
}

func (r *Result) Priorities() int {
	return r.priorities
}

func (r *Result) VisitTimes() VisitTimes {
	return r.visitTimes
}

func (r *Result) SetVisitTimes(visitTimes VisitTimes) {
	r.visitTimes = visitTimes
}

type VisitTimes struct {
	Arrivals   map[int]time.Time
	Departures map[int]time.Time
}

func NewVisitTimes(size int) VisitTimes {
	return VisitTimes{
		Arrivals:   make(map[int]time.Time, size),
		Departures: make(map[int]time.Time, size),
	}
}
