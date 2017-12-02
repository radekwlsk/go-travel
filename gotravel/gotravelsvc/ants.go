package gotravelsvc

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"time"
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

type Ant struct {
	trip          *Trip
	places        Places
	visitTimes    VisitTimes
	startPlace    *TripPlace
	endPlace      *TripPlace
	n             int
	bestPath      Path
	path          Path
	at            int
	used          Used
	currentTime   time.Time
	totalTime     time.Duration
	totalDistance float64
	distances     *DistanceMatrix
	times         *TravelTimeMatrix
	pheromones    *PheromonesMatrix
	random        *rand.Rand
	resultChannel chan Result
}

func NewAnt(
	trip *Trip,
	distances *DistanceMatrix,
	times *TravelTimeMatrix,
	pheromones *PheromonesMatrix,
	resultChannel chan Result,
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

func (a *Ant) SetPheromones(p *PheromonesMatrix) {
	a.pheromones = p
}

func (a *Ant) BestPath() *Path {
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
	a.pheromones.IntensifyAlong(&a.path, boost)
	a.resultChannel <- Result{a.path, a.totalTime, a.sumPriorities(), a.visitTimes}
}

func (a *Ant) FindFoodIterations(iterations, boost int) {
	var bestResult = Result{
		path:       NewDummyPath(),
		time:       time.Duration(math.MaxInt64),
		priorities: 0,
	}

	for i := 0; i < iterations; i++ {
		err := a.generatePath()
		if err != nil && err != ErrTripEnded {
			panic(err.Error())
		}

		sumPriorities := a.sumPriorities()

		if sumPriorities >= bestResult.priorities && a.totalTime <= bestResult.time {
			a.pheromones.IntensifyAlong(&a.path, boost)
			bestResult = Result{
				path:       a.path,
				time:       a.totalTime,
				priorities: sumPriorities,
			}
		}

		if err == ErrTripEnded {
			break
		}

		a.pheromones.Evaporate(boost, iterations)
	}

	bestResult.visitTimes = a.visitTimes
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
	a.places = NewPlaces(a.trip.Places)
	a.setStart()
	a.endPlace = a.trip.EndPlace
	a.before()
	a.random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (a *Ant) setStep(i int, place *TripPlace) {
	dur := a.times.At(a.at, place.Index, a.currentTime)
	stay := time.Duration(place.Place.StayDuration) * time.Minute
	dist := a.distances.At(a.at, place.Index)
	a.currentTime = a.currentTime.Add(dur)
	a.visitTimes.Arrivals[place.Index] = a.currentTime
	a.currentTime = a.currentTime.Add(stay)
	a.visitTimes.Departures[place.Index] = a.currentTime
	a.totalTime += dur
	a.totalTime += stay
	a.totalDistance += dist
	a.path.Set(i, place.Index)
	a.at = place.Index
	a.used[a.at] = true
}

func (a *Ant) isUsed(place *TripPlace) bool {
	return a.used[place.Index]
}

func (a *Ant) before() {
	a.path = NewPath(a.n, a.startPlace == a.endPlace)
	a.visitTimes = NewVisitTimes(a.n)
	a.used = make(Used, a.n)
	a.setStep(0, a.startPlace)
	a.bestPath = Path{
		len:  a.path.len,
		loop: a.path.loop,
	}
	copy(a.bestPath.path, a.path.path)
	a.currentTime = a.trip.TripStart
	a.totalTime = time.Duration(0)
	a.totalDistance = 0
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
			dist := a.distances.At(a.at, next.Index)
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

//func (a *Ant) endTrip() {
//	if a.path.loop {
//		dur := a.times.At(a.at, a.endPlace.Index, a.currentTime)
//		dist := a.distances.At(a.at, place.Index)
//	}
//}

func (a *Ant) pickNextPlace(i int) (place *TripPlace, err error) {
	if final := i == a.path.len-1; final && a.endPlace != nil && a.endPlace != a.startPlace {
		return a.endPlace, nil
	}

	var available []*TripPlace
	for _, p := range a.places {
		if !a.isUsed(p) && p != a.endPlace {
			available = append(available, p)
		}
	}
	var reachable []*TripPlace
	var pheromones []float64
	var fitness []float64
	for _, p := range available {
		ok, _ := a.isReachable(p)
		if ok {
			reachable = append(reachable, p)
			pheromones = append(pheromones, a.pheromones.At(a.at, p.Index))
			// TODO: fitness based on priority and travel time
			fitness = append(fitness, 1.0/a.times.AtAs(a.at, p.Index, a.currentTime, time.Minute))
		}
	}
	if len(reachable) == 0 {
		if a.path.loop {
			return a.endPlace, ErrMustReturnToStart
		} else if a.endPlace != nil {
			return a.endPlace, ErrMustReachEndPlace
		}
		return nil, ErrTripEnded
	}
	var total float64
	for i := range reachable {
		total += pheromones[i] * fitness[i]
	}
	for {
		for _, r := range reachable {
			// TODO: next place pickup based on some cost function
			return r, nil
			//if a.random.Float64() >= (pheromones[i]*fitness[i])/total {
			//	return r, nil
			//}
		}
	}
}

func (a *Ant) isReachable(place *TripPlace) (ok bool, err error) {
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
	for i := range a.path.path {
		sum += a.places[i].Place.Priority
	}
	return
}
