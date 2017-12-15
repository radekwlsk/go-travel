package ants

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelservice/trip"

	"github.com/gonum/floats"
)

var (
	ErrPlaceClosed         = errors.New("place closed at that day")
	ErrPlaceClosesTooEarly = errors.New("place closes too early")
	ErrTripEndsTooEarly    = errors.New("trip time ends before place departure")
	ErrCantReachEndPlace   = errors.New("can't reach set end place in time")
	ErrTripEnded           = errors.New("no place reachable before trip time end")
	ErrMustReturnToStart   = errors.New("must return to start place before trip ends")
	ErrMustReachEndPlace   = errors.New("must get to end place before trip ends")
)

type Used map[int]bool

type Ant struct {
	trip          *trip.Trip
	visitTimes    VisitTimes
	startPlace    *trip.Place
	endPlace      *trip.Place
	n             int
	path          trip.Path
	at            int
	used          Used
	currentTime   time.Time
	totalTime     time.Duration
	totalDistance int64
	distances     *TimesMappedDistancesMatrix
	durations     *TimesMappedDurationsMatrix
	pheromones    *PheromonesMatrix
	random        *rand.Rand
	resultChannel chan Result
}

func NewAnt(
	trip *trip.Trip,
	distances *TimesMappedDistancesMatrix,
	durations *TimesMappedDurationsMatrix,
	pheromones *PheromonesMatrix,
	resultChannel chan Result,
) (a *Ant) {
	a = &Ant{
		trip:          trip,
		distances:     distances,
		durations:     durations,
		pheromones:    pheromones,
		resultChannel: resultChannel,
	}
	a.init()

	return a
}

func (a *Ant) SetPheromones(p *PheromonesMatrix) {
	a.pheromones = p
}

func (a *Ant) FindFood() {
	err := a.before()
	switch err {
	case ErrTripEnded:
		a.resultChannel <- NewResult(
			a.path,
			a.totalTime,
			a.totalDistance,
			a.sumPriorities(),
			a.visitTimes,
		)
	case nil:
		err = a.generatePath()
		if err != nil && err != ErrTripEnded {
			panic(err.Error())
		}
		a.resultChannel <- NewResult(
			a.path,
			a.totalTime,
			a.totalDistance,
			a.sumPriorities(),
			a.visitTimes,
		)
	default:
		panic(err.Error())
	}
}

func (a *Ant) setStart() error {
	if a.trip.StartPlace == nil {
		var reachable []*trip.Place

		for _, p := range a.trip.Places {
			a.at = p.Index
			if ok, _ := a.placeReachable(p); ok && p != a.endPlace {
				reachable = append(reachable, p)
			}
		}
		if n := len(reachable); n > 0 {
			i := a.random.Intn(n)
			a.startPlace = reachable[i]
		} else if a.endPlace != nil {
			a.startPlace = a.endPlace
		} else {
			return ErrTripEnded
		}
	} else {
		a.startPlace = a.trip.StartPlace
	}
	return nil
}

func (a *Ant) init() {
	a.n = len(a.trip.Places)
	a.random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (a *Ant) setStep(i int, place *trip.Place) {
	var dist int64
	var dur time.Duration
	arrival, departure, err := a.placeArrivalDeparture(place, i == 0)
	if err != nil {
		panic(err.Error())
	}
	if i > 0 {
		dist = a.distances.At(a.at, place.Index, a.currentTime)
		dur = arrival.Sub(a.currentTime)
		a.path.SetStep(i, place.Index, dur, dist)
		a.totalTime += dur
		a.currentTime = arrival
		a.totalDistance += dist
	} else {
		a.path.Set(0, place.Index)
	}
	if place != a.startPlace || i == 0 {
		a.visitTimes.Arrivals[place.Index] = arrival
		a.totalTime += departure.Sub(a.currentTime)
		a.currentTime = departure
		a.visitTimes.Departures[place.Index] = departure
	}
	a.at = place.Index
	a.used[a.at] = true
}

func (a *Ant) isUsed(place *trip.Place) bool {
	return a.used[place.Index]
}

func (a *Ant) before() error {
	a.endPlace = a.trip.EndPlace
	a.visitTimes = NewVisitTimes(a.n)
	a.used = make(Used, a.n)
	a.currentTime = a.trip.TripStart
	a.totalTime = time.Duration(0)
	a.totalDistance = 0
	if err := a.setStart(); err != nil {
		return err
	}
	a.path = trip.NewPath(a.n, a.startPlace == a.endPlace)
	a.setStep(0, a.startPlace)
	return nil
}

func (a *Ant) generatePath() error {
	for i := 1; i < a.n; i++ {
		switch next, err := a.pickNextPlace(); err {
		case ErrMustReachEndPlace:
			a.setStep(i, next)
			if i+1 < a.path.Size()-1 {
				a.path.Cut(i + 1)
			}
			return ErrTripEnded
		case ErrTripEnded:
			a.path.Cut(i)
			return ErrTripEnded
		case ErrMustReturnToStart:
			if i < a.path.Size()-1 {
				a.path.Cut(i)
			}
			a.setStep(i, next)
			return ErrTripEnded
		case nil:
			a.setStep(i, next)
		default:
			panic("unexpected error returned from Ant.pickNextPlace()")
		}
	}
	switch a.endPlace {
	case nil:
		return ErrTripEnded
	case a.startPlace:
		a.setStep(a.n, a.startPlace)
		return ErrTripEnded
	default:
		panic("end place != start place left to visit after loop")
	}
}

func (a *Ant) pickNextPlace() (place *trip.Place, err error) {
	var available []*trip.Place
	for _, p := range a.trip.Places {
		if !a.isUsed(p) && p != a.endPlace {
			available = append(available, p)
		}
	}
	var reachable []*trip.Place
	var pheromones []float64

	for _, p := range available {
		if ok, _ := a.placeReachable(p); ok {
			reachable = append(reachable, p)
			pheromone := a.pheromones.At(a.at, p.Index)
			pheromones = append(pheromones, pheromone)
		}
	}
	l := len(reachable)
	if l == 0 {
		if a.startPlace == a.endPlace {
			return a.endPlace, ErrMustReturnToStart
		} else if a.endPlace != nil {
			return a.endPlace, ErrMustReachEndPlace
		}
		return nil, ErrTripEnded
	}
	total := floats.Sum(pheromones)
	for {
		for i := range a.random.Perm(l) {
			if a.random.Float64() <= pheromones[i]/total {
				return reachable[i], nil
			}
		}
	}
}

func (a *Ant) placeArrivalDeparture(place *trip.Place, first bool) (arrival, departure time.Time, err error) {
	var opn, cls time.Time
	if first {
		arrival = a.currentTime
	} else {
		dur := a.durations.At(a.at, place.Index, a.currentTime)
		arrival = a.currentTime.Add(dur)
	}

	if !first && place == a.startPlace {
		return arrival, arrival, nil
	}

	departure = arrival.Add(time.Duration(place.StayDuration) * time.Minute)
	oc := place.Details.OpeningHoursPeriods[a.currentTime.Weekday()]
	if oc.Open == "" && oc.Close == "" {
		return arrival, departure, ErrPlaceClosed
	}
	{
		o := oc.Open
		y, m, d := arrival.In(place.Details.Location).Date()
		hh, _ := strconv.Atoi(o[:2])
		mm, _ := strconv.Atoi(o[2:])
		opn = time.Date(y, m, d, hh, mm, 0, 0, place.Details.Location).In(arrival.Location())
	}
	if opn.After(arrival) {
		departure = opn.Add(time.Duration(place.StayDuration) * time.Minute)
	}
	{
		c := oc.Close
		y, m, d := departure.In(place.Details.Location).Date()
		hh, _ := strconv.Atoi(c[:2])
		mm, _ := strconv.Atoi(c[2:])
		cls = time.Date(y, m, d, hh, mm, 0, 0, place.Details.Location).In(arrival.Location())
	}
	if cls.Before(departure) {
		return arrival, departure, ErrPlaceClosesTooEarly
	}
	if a.trip.TripEnd.Before(departure) {
		return arrival, departure, ErrTripEndsTooEarly
	}
	return
}

func (a *Ant) placeReachable(place *trip.Place) (ok bool, err error) {
	if place.Details.PermanentlyClosed {
		return false, ErrPlaceClosed
	}

	_, dprt, err := a.placeArrivalDeparture(place, a.currentTime.Equal(a.trip.TripStart))
	if err != nil {
		return false, err
	}

	if a.endPlace != nil {
		fin := dprt.Add(a.durations.At(place.Index, a.endPlace.Index, dprt))
		if a.endPlace != a.startPlace {
			stay := time.Duration(a.endPlace.StayDuration) * time.Minute
			fin = fin.Add(stay)
		}
		if a.trip.TripEnd.Before(fin) {
			return false, ErrCantReachEndPlace
		}
	}
	return true, nil
}

func (a *Ant) sumPriorities() (sum int) {
	for _, i := range a.path.Path() {
		sum += a.trip.Places[i].Priority
	}
	return
}
