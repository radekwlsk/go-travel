package ants

import (
	"math"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelservice/trip"
)

type Result struct {
	path       trip.Path
	time       time.Duration
	distance   int64
	priorities int
	visitTimes VisitTimes
}

func NewResult(path trip.Path, dur time.Duration, dist int64, prio int, times VisitTimes) Result {
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
		path:       trip.NewDummyPath(),
		time:       time.Duration(math.MaxInt64),
		distance:   math.MaxInt64,
		priorities: 0,
		visitTimes: VisitTimes{},
	}
}

func (r *Result) BetterThan(o Result) bool {
	if r.priorities < o.priorities {
		return false
	}
	if r.priorities > o.priorities {
		return true
	}

	if r.priorities == o.priorities {
		if r.path.Size() > o.path.Size() {
			return true
		}
		if r.path.Size() == o.path.Size() && r.time < o.time {
			return true
		}
	}
	return false
}

func (r *Result) Path() trip.Path {
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
