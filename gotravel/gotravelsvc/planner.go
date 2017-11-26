package gotravelsvc

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"googlemaps.github.io/maps"
)

const (
	Iterations = 1
	Boost      = 5
	Ants       = 1
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
	var pherMutex = &sync.Mutex{}

	{
		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		length := len(planner.trip.Places)

		times = NewTravelTimeMatrix(length)
		distances = NewDistanceMatrix(length)

		for r := 1; r < length; r++ {
			for c := 1; c < length && r != c; c++ {
				times.Set(r, c, time.Duration(random.Intn(20)+10)*time.Minute)
				distances.Set(r, c, float64(random.Intn(10)+5))
			}
		}

		pheromones = NewPheromonesMatrix(length, float64(Boost))
	}

	var bestPath = NewDummyPath()
	var bestTime = time.Duration(math.MaxInt64)

	for i := 1; i <= Ants; i++ {
		ant := NewAnt(planner.trip, distances, times, pheromones, pherMutex)
		err = ant.FindFood(Iterations, Boost)
		if err != nil {
			return indexes, err
		}
		if ant.totalTime <= bestTime {
			bestPath = *ant.BestPath()
			bestTime = ant.totalTime
		}
	}

	return bestPath.PathIndexes(), err
}
