package planner

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/planner/types"
	traveltypes "github.com/afrometal/go-travel/gotravel/gotravelsvc/types"
	"github.com/kataras/iris/core/errors"
	"github.com/kr/pretty"
	"googlemaps.github.io/maps"
)

const (
	Iterations = 50
	Boost      = 5
	Ants       = 50
)

type Planner struct {
	client    *maps.Client
	trip      *traveltypes.Trip
	times     *types.TravelTimeMatrix
	distances *types.DistanceMatrix
}

func NewPlanner(c *maps.Client, t *traveltypes.Trip) *Planner {
	return &Planner{
		client: c,
		trip:   t,
	}
}

func (planner *Planner) Evaluate() (steps []traveltypes.Step, err error) {
	var ants [Ants]*Ant
	var length int
	var times *types.TravelTimeMatrix
	var distances *types.DistanceMatrix
	var pheromones *types.PheromonesMatrix
	var pherMutex = sync.Mutex{}
	var resultChannel = make(chan types.Result)

	length = len(planner.trip.Places)
	times, distances, err = planner.getTimesAndDistances()

	if err != nil {
		return nil, err
	}

	pheromones = types.NewPheromonesMatrix(length, float64(Boost), pherMutex)

	var bestResult = types.NewEmptyResult()

	for i := 0; i < Ants; i++ {
		ants[i] = NewAnt(planner.trip, distances, times, pheromones, resultChannel)
	}
	for i := 0; i < Iterations; i++ {
		//if i%int(float64(Iterations)/10.0) == 0 {
		//	pheromones = NewPheromonesMatrix(length, float64(Boost), pherMutex)
		//}
		for i := 0; i < Ants; i++ {
			go ants[i].FindFood(Boost)
		}
		for i := 0; i < Ants; i++ {
			result := <-resultChannel
			if result.Priorities() > bestResult.Priorities() ||
				(result.Priorities() == bestResult.Priorities() && result.Time() < bestResult.Time()) {
				pretty.Printf(
					"better result! time: %.2f minutes, priorities: %d\n",
					float64(result.Time()/time.Minute),
					result.Priorities(),
				)
				pheromones.IntensifyAlong(result.Path(), Boost)
				bestResult = result
			}
		}
		pheromones.Evaporate(Boost, Iterations)
	}

	for _, place := range planner.trip.Places {
		place.Arrival = bestResult.VisitTimes().Arrivals[place.Index]
		place.Departure = bestResult.VisitTimes().Departures[place.Index]
	}
	planner.trip.TripEnd = planner.trip.TripStart.Add(bestResult.Time())
	planner.trip.TotalDistance = bestResult.Distance()

	return bestResult.Path().Steps, err
}

func (planner *Planner) getTimesAndDistances() (times *types.TravelTimeMatrix, distances *types.DistanceMatrix, err error) {
	length := len(planner.trip.Places)
	currentTime := planner.trip.TripStart
	var checkedTimes []time.Time
	timeDelta := time.Duration(4 * time.Hour)
	for !currentTime.After(planner.trip.TripEnd) {
		checkedTimes = append(checkedTimes, currentTime)
		currentTime = currentTime.Add(timeDelta)
	}
	times = types.NewTravelTimeMatrix(length, checkedTimes)
	distances = types.NewDistanceMatrix(length, checkedTimes)
	destinationAddresses := make([]string, length)
	originAddresses := make([]string, length)
	for _, place := range planner.trip.Places {
		destinationAddresses[place.Index] = place.Details.FormattedAddress
		originAddresses[place.Index] = place.Details.FormattedAddress
	}
	for _, t := range checkedTimes {
		r := &maps.DistanceMatrixRequest{
			Origins:       originAddresses,
			Destinations:  destinationAddresses,
			DepartureTime: strconv.FormatInt(t.Unix(), 10),
		}
		var resp *maps.DistanceMatrixResponse
		resp, err := planner.client.DistanceMatrix(context.Background(), r)
		if err != nil {
			return times, distances, err
		}
		for i, row := range resp.Rows {
			for j, element := range row.Elements {
				if i != j {
					if element.Status == "OK" {
						times.Set(i, j, t, element.DurationInTraffic)
						distances.Set(i, j, t, int64(element.Distance.Meters))
					} else {
						return times, distances, errors.New(fmt.Sprintf(
							"could not get distances between %s and %s at %s",
							originAddresses[i],
							destinationAddresses[j],
							t.String(),
						))
					}
				}
			}
		}
	}

	return times, distances, nil
}
