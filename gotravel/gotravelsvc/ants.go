package gotravelsvc

import (
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/kataras/iris/core/errors"
)

type Ant struct {
	trip          *Trip
	places        Places
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
	pherMutex     *sync.Mutex
	random        *rand.Rand
	bestChannel   chan Path
}

func NewAnt(
	trip *Trip,
	distances *DistanceMatrix,
	times *TravelTimeMatrix,
	pheromones *PheromonesMatrix,
	pherMutex *sync.Mutex,
) (a *Ant) {
	a = &Ant{
		trip:       trip,
		distances:  distances,
		times:      times,
		pheromones: pheromones,
		pherMutex:  pherMutex,
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

func (a *Ant) FindFood(iterations int, boost int) error {
	bestTime := time.Duration(math.MaxInt64)

	for i := 0; i < iterations; i++ {
		err := a.generatePath()
		if err != nil {
			return err
		}
		totalTime := a.totalTime

		if totalTime <= bestTime {
			a.pheromones.IntensifyAlong(&a.path, boost, a.pherMutex)
			a.bestPath = Path{
				path: a.path.path,
				len:  a.path.len,
				loop: a.path.loop,
			}
		}

		a.pheromones.Evaporate(boost, iterations, a.pherMutex)
	}

	return nil
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
	a.path.Set(i, place.Index)
	dur := a.times.At(a.at, place.Index)
	stay := time.Duration(place.Place.StayDuration) * time.Minute
	dist := a.distances.At(a.at, place.Index)
	a.currentTime = a.currentTime.Add(dur).Add(stay)
	a.totalTime += dur
	a.totalTime += stay
	a.totalDistance += dist
	a.at = place.Index
	a.used[a.at] = true
}

func (a *Ant) isUsed(place *TripPlace) bool {
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

func (a *Ant) generatePath() error {
	a.setStart()
	a.before()

	for i := 1; i < a.n; i++ {
		next, err := a.pickNextPlace(i)
		if err != nil {
			return err
		}
		a.setStep(i, next)
	}

	return nil
}

func (a *Ant) pickNextPlace(i int) (place *TripPlace, err error) {
	final := i == a.path.len-1
	var available []*TripPlace
	for _, p := range a.places {
		if !a.isUsed(p) {
			available = append(available, p)
		}
	}
	var reachable []*TripPlace
	var pheromones []float64
	var fitness []float64
	for _, p := range available {
		_, err := a.isReachable(p, final)
		if err != nil {
			reachable = append(reachable, p)
			pheromones = append(pheromones, a.pheromones.At(a.at, p.Index))
			fitness = append(fitness, 1.0/a.times.AtAs(a.at, p.Index, time.Minute))
		}
	}
	if len(reachable) == 0 {
		return nil, errors.New("no reachable places")
	}
	var total float64
	for i := range reachable {
		total += pheromones[i] * fitness[i]
	}
	for {
		for _, r := range reachable {
			return r, nil
			//if a.random.Float64() >= (pheromones[i]*fitness[i])/total {
			//	return r, nil
			//}
		}
	}
}

func (a *Ant) isReachable(place *TripPlace, final bool) (ok bool, err error) {
	dur := a.times.At(a.at, place.Index)
	var arvl, dprt, opn, cls time.Time
	{
		arvl = a.currentTime.Add(dur)
		dprt = a.currentTime.Add(dur).Add(time.Duration(place.Place.StayDuration) * time.Minute)
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
		return false, errors.New("place opens too late")
	} else {
		place.Arrival = arvl
	}
	if cls.Before(dprt) {
		return true, errors.New("place closes too early")
	} else {
		place.Departure = dprt
	}

	if final && a.path.loop {
		fin := dprt.Add(a.times.At(place.Index, a.startPlace.Index))
		if a.trip.TripEnd.Before(fin) {
			return true, errors.New("can't get back to start in time")
		}
		return true, nil
	} else if a.trip.TripEnd.Before(dprt) {
		return true, errors.New("trip time ends before departure")
	} else {
		return true, nil
	}
}
