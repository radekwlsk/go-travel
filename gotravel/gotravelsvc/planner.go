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
	Iterations = 20
	Boost      = 5
	Ants       = 10
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
	var times *TravelTimeMatrix
	var distances *DistanceMatrix
	var pheromones *PheromonesMatrix
	var pherMutex = sync.Mutex{}
	var resultChannel = make(chan Result)

	{
		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		length := len(planner.trip.Places)

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

	var bestPath = NewDummyPath()
	var bestTime = time.Duration(math.MaxInt64)

	var ants [Ants]*Ant

	for i := 0; i < Ants; i++ {
		ants[i] = NewAnt(planner.trip, distances, times, pheromones, resultChannel)
	}
	for i := 0; i < Iterations; i++ {
		for i := 0; i < Ants; i++ {
			go ants[i].FindFood(Boost)
		}
		for i := 0; i < Ants; i++ {
			result := <-resultChannel
			if result.time <= bestTime {
				pretty.Printf(
					"better result! time: %.2f minutes, priorities: %d\n",
					float64(result.time/time.Minute),
					result.priorities,
				)
				pretty.Println(result.path.PathIndexes())
				bestPath = result.path
				bestTime = result.time
			}
		}
		pheromones.Evaporate(Boost, Iterations)
	}

	return bestPath.PathIndexes(), err
}
