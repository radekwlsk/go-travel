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
	at            int
	used          Used
	currentTime   time.Time
	totalTime     time.Duration
	totalDistance Distance
	distances     *DistanceMatrix
	times         *TravelTimeMatrix
	pheromones    *PheromonesMatrix
	random        *rand.Rand
	bestChannel   chan Path
}

func NewAnt(trip *gotravelsvc.Trip, distances *DistanceMatrix, times *TravelTimeMatrix) (a *Ant) {
	a.trip = trip
	a.distances = distances
	a.times = times
	a.init()

	return a
}

func (a *Ant) SetPheromones(p *PheromonesMatrix) {
	a.pheromones = p
}

func (a *Ant) Pheromones() *PheromonesMatrix {
	return a.pheromones
}

func (a *Ant) BestPath() *Path {
	return &a.bestPath
}

func (a *Ant) setStart() {
	if a.trip.StartPlace == nil {
		i := a.random.Intn(a.n)
		a.startPlace = a.places[i]
	} else {
		a.startPlace = a.trip.StartPlace
	}
}

func (a *Ant) init() {
	a.n = len(a.trip.Places)
	a.places = NewPlaces(a.trip.Places)
	a.setStart()
	a.endPlace = a.trip.EndPlace
	a.before()
	a.pheromones = NewPheromonesMatrix(a.n)
	a.random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (a *Ant) setStep(step int, place *gotravelsvc.TripPlace) {
	a.path.Set(step, place.Index)
	a.used[place.Index] = true
	a.at = place.Index
}

func (a *Ant) isUsed(place *gotravelsvc.TripPlace) bool {
	return a.used[place.Index]
}

func (a *Ant) before() {
	a.path = NewPath(a.n, a.startPlace == a.endPlace)
	a.used = make(Used, a.n)
	a.setStep(0, a.startPlace)
	a.bestPath = a.path
	a.currentTime = a.trip.TripStart
	a.totalTime = time.Duration(0)
	a.totalDistance = 0
}

func (a *Ant) generatePath() Path {
	a.setStart()
	a.before()

	for i := 1; i < a.n; i++ {
		sumWeight := a.sumWeight()
		rndValue := a.random.Float64() * sumWeight
		next := a.findSumWeight(rndValue)
		a.setStep(i, next)
	}

	return a.path
}

func (a *Ant) findSumWeight(f float64) *gotravelsvc.TripPlace {
	panic("Not implemented")
}

func (a *Ant) sumWeight() float64 {
	panic("Not implemented")
}
