package ants

import (
	"github.com/afrometal/go-travel/gotravel/gotravelsvc"
	"math/rand"
	"time"
)

type Ant struct {
	trip          *gotravelsvc.Trip
	places        Places
	startPlace    *gotravelsvc.TripPlace
	endPlace      *gotravelsvc.TripPlace
	n             int
	bestPath      Path
	path          Path
	used          Used
	currentTime   time.Time
	totalTime     time.Duration
	totalDistance Distance
	distances     *DistanceMatrix
	times         *TravelTimeMatrix
	pheromones    *PheromonesMatrix
	random        rand.Rand
}

func (a *Ant) reset() {
	if a.trip.StartPlace == nil {
		i := a.random.Intn(a.n)
		a.startPlace = a.places[i]
	} else {
		a.startPlace = a.trip.StartPlace
	}
	a.path = NewPath(a.n, a.startPlace == a.endPlace)
	a.path.Set(0, a.startPlace.Index)
	a.used = make(Used, a.n)
	a.used[0] = true
	a.bestPath = a.path
	a.currentTime = a.trip.TripStart
	a.totalTime = time.Duration(0)
	a.totalDistance = 0
}

func (a *Ant) generatePath() {
	a.reset()
}
