package planner

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/planner/types"
	traveltypes "github.com/afrometal/go-travel/gotravel/gotravelsvc/types"

	"github.com/gonum/floats"
)

var (
	ErrPlaceOpensTooLate   = errors.New("place opens too late")
	ErrPlaceClosesTooEarly = errors.New("place closes too early")
	ErrTripEndsTooEarly    = errors.New("trip time ends before place departure")
	ErrCantReachEndPlace   = errors.New("can't reach set end place in time")
	ErrTripEnded           = errors.New("no place reachable before trip time end")
	ErrMustReturnToStart   = errors.New("must return to start place before trip ends")
	ErrMustReachEndPlace   = errors.New("must get to end place before trip ends")
)

type Used map[int]bool
type Places map[int]*traveltypes.TripPlace

func NewPlaces(tps []*traveltypes.TripPlace) Places {
	places := make(Places, len(tps))
	for _, tp := range tps {
		places[tp.Index] = tp
	}
	return places
}

type Ant struct {
	trip          *traveltypes.Trip
	places        Places
	visitTimes    types.VisitTimes
	startPlace    *traveltypes.TripPlace
	endPlace      *traveltypes.TripPlace
	n             int
	bestPath      traveltypes.Path
	path          traveltypes.Path
	at            int
	used          Used
	currentTime   time.Time
	totalTime     time.Duration
	totalDistance int64
	distances     *types.TimesMappedDistancesMatrix
	times         *types.TimesMappedDurationsMatrix
	pheromones    *types.PheromonesMatrix
	random        *rand.Rand
	resultChannel chan types.Result
}

func NewAnt(
	trip *traveltypes.Trip,
	distances *types.TimesMappedDistancesMatrix,
	times *types.TimesMappedDurationsMatrix,
	pheromones *types.PheromonesMatrix,
	resultChannel chan types.Result,
) (a *Ant) {
	a = &Ant{
		trip:          trip,
		distances:     distances,
		times:         times,
		pheromones:    pheromones,
		resultChannel: resultChannel,
	}
	a.init()

	return a
}

func (a *Ant) SetPheromones(p *types.PheromonesMatrix) {
	a.pheromones = p
}

func (a *Ant) BestPath() *traveltypes.Path {
	return &a.bestPath
}

func (a *Ant) TotalTime() time.Duration {
	return a.totalTime
}

func (a *Ant) FindFood(boost int) {
	err := a.generatePath()
	if err != nil && err != ErrTripEnded {
		panic(err.Error())
	}
	a.resultChannel <- types.NewResult(
		a.path,
		a.totalTime,
		a.totalDistance,
		a.sumPriorities(),
		a.visitTimes,
	)

}

func (a *Ant) FindFoodIterations(iterations, boost int) {
	var bestResult = types.NewEmptyResult()

	for i := 0; i < iterations; i++ {
		err := a.generatePath()
		if err != nil && err != ErrTripEnded {
			panic(err.Error())
		}

		sumPriorities := a.sumPriorities()

		if sumPriorities >= bestResult.Priorities() && a.totalTime <= bestResult.Time() {
			a.pheromones.IntensifyAlong(a.path, boost)
			bestResult = types.NewResult(a.path, a.totalTime, a.totalDistance, sumPriorities, types.VisitTimes{})
		}

		if err == ErrTripEnded {
			break
		}

		a.pheromones.Evaporate(boost, iterations)
	}

	bestResult.SetVisitTimes(a.visitTimes)
	a.resultChannel <- bestResult
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
	a.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	a.places = NewPlaces(a.trip.Places)
	a.setStart()
	a.endPlace = a.trip.EndPlace
	a.before()
}

func (a *Ant) setStep(i int, place *traveltypes.TripPlace) {
	var dur, stay time.Duration
	var dist int64
	if i > 0 {
		dur = a.times.At(a.at, place.Index, a.currentTime)
		a.currentTime = a.currentTime.Add(dur)
		a.totalTime += dur
		dist = a.distances.At(a.at, place.Index, a.currentTime)
		a.totalDistance += dist
		a.path.SetStep(i, place.Index, dur, dist)
	} else {
		a.path.Set(i, place.Index)
	}
	stay = time.Duration(place.Place.StayDuration) * time.Minute
	a.visitTimes.Arrivals[place.Index] = a.currentTime
	a.currentTime = a.currentTime.Add(stay)
	a.visitTimes.Departures[place.Index] = a.currentTime
	a.totalTime += stay
	a.at = place.Index
	a.used[a.at] = true
}

func (a *Ant) isUsed(place *traveltypes.TripPlace) bool {
	return a.used[place.Index]
}

func (a *Ant) before() {
	a.path = traveltypes.NewPath(a.n, a.startPlace == a.endPlace)
	a.visitTimes = types.NewVisitTimes(a.n)
	a.used = make(Used, a.n)
	a.currentTime = a.trip.TripStart
	a.totalTime = time.Duration(0)
	a.totalDistance = 0
	a.setStep(0, a.startPlace)
	a.bestPath = a.path
}

func (a *Ant) generatePath() error {
	a.setStart()
	a.before()

	for i := 1; i < a.n; i++ {
		switch next, err := a.pickNextPlace(i); err {
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
			dur := a.times.At(a.at, next.Index, a.currentTime)
			dist := a.distances.At(a.at, next.Index, a.currentTime)
			a.currentTime = a.currentTime.Add(dur)
			a.totalTime += dur
			a.totalDistance += dist
			a.at = next.Index
			return ErrTripEnded
		case nil:
			a.setStep(i, next)
		default:
			panic("unexpected error returned from Ant.pickNextPlace()")
		}
	}
	return nil
}

func (a *Ant) pickNextPlace(i int) (place *traveltypes.TripPlace, err error) {
	if final := i == a.path.Size()-1; final && a.endPlace != nil && a.endPlace != a.startPlace {
		return a.endPlace, nil
	}

	var available []*traveltypes.TripPlace
	for _, p := range a.places {
		if !a.isUsed(p) && p != a.endPlace {
			available = append(available, p)
		}
	}
	var reachable []*traveltypes.TripPlace
	var pheromones []float64

	for _, p := range available {
		if ok, _ := a.isReachable(p); ok {
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

func (a *Ant) isReachable(place *traveltypes.TripPlace) (ok bool, err error) {
	dur := a.times.At(a.at, place.Index, a.currentTime)
	var arvl, dprt, opn, cls time.Time
	{
		arvl = a.currentTime.Add(dur)
		dprt = arvl.Add(time.Duration(place.Place.StayDuration) * time.Minute)
		oc := place.Details.OpeningHoursPeriods[a.currentTime.Weekday()]
		o := oc.Open.Time
		y, m, d := arvl.In(place.Details.Location).Date()
		hh, _ := strconv.Atoi(o[:2])
		mm, _ := strconv.Atoi(o[2:])
		opn = time.Date(y, m, d, hh, mm, 0, 0, place.Details.Location).In(arvl.Location())
		c := oc.Close.Time
		y, m, d = dprt.In(place.Details.Location).Date()
		hh, _ = strconv.Atoi(c[:2])
		mm, _ = strconv.Atoi(c[2:])
		cls = time.Date(y, m, d, hh, mm, 0, 0, place.Details.Location).In(arvl.Location())
	}

	if opn.After(arvl) {
		return false, ErrPlaceOpensTooLate
	}
	if cls.Before(dprt) {
		return false, ErrPlaceClosesTooEarly
	}
	if a.trip.TripEnd.Before(dprt) {
		return false, ErrTripEndsTooEarly
	}

	if a.endPlace != nil {
		fin := dprt.Add(a.times.At(place.Index, a.endPlace.Index, dprt))
		if a.trip.TripEnd.Before(fin) {
			return false, ErrCantReachEndPlace
		}
	}
	return true, nil
}

func (a *Ant) sumPriorities() (sum int) {

	for i := range a.path.Path() {
		sum += a.places[i].Place.Priority
	}
	return
}
