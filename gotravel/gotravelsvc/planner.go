package gotravelsvc

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/kr/pretty"
	"googlemaps.github.io/maps"
)

const (
	Iterations = 50
	Boost      = 5
	Ants       = 20
)

type Planner struct {
	client *maps.Client
	trip   *Trip
}

func NewPlanner(c *maps.Client, t *Trip) *Planner {
	return &Planner{client: c, trip: t}
}

func (planner *Planner) Evaluate() (indexes []int, err error) {
	//var ants []*Ant
	var length int
	var times *TravelTimeMatrix
	var distances *DistanceMatrix
	var pheromones *PheromonesMatrix
	var pherMutex = sync.Mutex{}
	var resultChannel = make(chan Result)

	{
		random := rand.New(rand.NewSource(time.Now().UnixNano()))

		length = len(planner.trip.Places)
		times = NewTravelTimeMatrix(length)
		distances = NewDistanceMatrix(length)

		for r := 1; r < length; r++ {
			for c := 1; c < length && r != c; c++ {
				times.Set(r, c, time.Now(), time.Duration(random.Intn(10)+10)*time.Minute)
				distances.Set(r, c, float64(random.Intn(10)+5))
			}
		}

		pheromones = NewPheromonesMatrix(length, float64(Boost), pherMutex)
	}

	var bestResult = Result{
		path:       NewDummyPath(),
		time:       time.Duration(math.MaxInt64),
		priorities: 0,
	}

	var ants [Ants]*Ant

	for i := 0; i < Ants; i++ {
		ants[i] = NewAnt(planner.trip, distances, times, pheromones, resultChannel)
	}
	for i := 0; i < Iterations; i++ {
		if i%int(float64(Iterations)/10.0) == 0 {
			pheromones = NewPheromonesMatrix(length, float64(Boost), pherMutex)
			for i := 0; i < Ants; i++ {
				ants[i].SetPheromones(pheromones)
			}
		}
		for i := 0; i < Ants; i++ {
			go ants[i].FindFood(Boost)
		}
		for i := 0; i < Ants; i++ {
			result := <-resultChannel
			if result.priorities >= bestResult.priorities && result.time <= bestResult.time {
				pretty.Printf(
					"better result! time: %.2f minutes, priorities: %d\n",
					float64(result.time/time.Minute),
					result.priorities,
				)
				pretty.Println(result.path.PathIndexes())
				bestResult = result
			}
		}
		pheromones.Evaporate(Boost, Iterations)
	}

	for _, place := range planner.trip.Places {
		place.Arrival = bestResult.visitTimes.Arrivals[place.Index]
		place.Departure = bestResult.visitTimes.Departures[place.Index]
	}
	planner.trip.TripEnd = planner.trip.TripStart.Add(bestResult.time)

	return bestResult.path.PathIndexes(), err
}
