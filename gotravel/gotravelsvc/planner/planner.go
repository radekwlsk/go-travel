package planner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/planner/ants"
	"github.com/afrometal/go-travel/gotravel/gotravelsvc/trip"
	"googlemaps.github.io/maps"
)

const Iterations = 10000

type Planner struct {
	client *maps.Client
	trip   *trip.Trip
	ants   int
	boost  float64
}

func NewPlanner(c *maps.Client, t *trip.Trip) *Planner {
	return &Planner{
		client: c,
		trip:   t,
	}
}

func (planner *Planner) Evaluate() (err error) {
	var durations *ants.TimesMappedDurationsMatrix
	var distances *ants.TimesMappedDistancesMatrix
	var pheromones *ants.PheromonesMatrix
	var resultChannel = make(chan ants.Result)
	var swarm []*ants.Ant
	{
		var length int
		var priorities float64
		for _, p := range planner.trip.Places {
			priorities += float64(p.Priority)
			length++
		}
		planner.ants = int(math.Ceil(5.0 * math.Sqrt(float64(length))))
		planner.boost = priorities / float64(length)
		durations, distances, err = planner.durationsAndDistances()
		if err != nil {
			return err
		}
		swarm = make([]*ants.Ant, planner.ants)
		pheromones = ants.NewPheromonesMatrix(length, planner.boost, sync.Mutex{})
	}

	var bestResult = ants.NewEmptyResult()
	var results = make([]ants.Result, planner.ants)

	for i := 0; i < planner.ants; i++ {
		swarm[i] = ants.NewAnt(planner.trip, distances, durations, pheromones, resultChannel)
	}
	for i := 0; i < Iterations; i++ {
		for i := 0; i < planner.ants; i++ {
			go swarm[i].FindFood()
		}
		for i := 0; i < planner.ants; i++ {
			results[i] = <-resultChannel

			if results[i].BetterThan(bestResult) {
				fmt.Printf(
					"better result! time: %.2f minutes, priorities: %d\n",
					float64(results[i].Time()/time.Minute),
					results[i].Priorities(),
				)
				bestResult = results[i]
			}
		}
		pheromones.Evaporate(planner.boost, Iterations)
		bestP := float64(bestResult.Priorities())
		b := math.Pow(2.0*bestP, 2.0)
		pheromones.IntensifyAlong(bestResult.Path(), planner.boost)
		var wg sync.WaitGroup
		for _, r := range results {
			wg.Add(1)
			go func(rP float64, p trip.Path) {
				defer wg.Done()
				r := math.Pow(bestP+rP, 2.0) / b
				ph := planner.boost * r
				pheromones.IntensifyAlong(p, ph)
			}(float64(r.Priorities()), r.Path())
		}
		wg.Wait()
	}
	close(resultChannel)

	for _, place := range planner.trip.Places {
		place.Arrival = bestResult.VisitTimes().Arrivals[place.Index]
		place.Departure = bestResult.VisitTimes().Departures[place.Index]
	}
	planner.trip.TripEnd = planner.trip.TripStart.Add(bestResult.Time())
	planner.trip.TotalDistance = bestResult.Distance()

	path := bestResult.Path()

	if path.Size() > 0 {
		if planner.trip.StartPlace == nil {
			planner.trip.StartPlace = planner.trip.Places[path.At(0)]
		}
		if planner.trip.EndPlace == nil {
			planner.trip.EndPlace = planner.trip.Places[path.At(path.Size()-1)]
		}
	}

	planner.trip.Path = path.Path()
	planner.trip.Steps = path.Steps
	planner.trip.CreateSchedule()

	return err
}

func (planner *Planner) durationsAndDistances() (
	durations *ants.TimesMappedDurationsMatrix,
	distances *ants.TimesMappedDistancesMatrix,
	err error,
) {
	length := len(planner.trip.Places)
	currentTime := planner.trip.TripStart
	var checkedTimes []time.Time
	var timeDelta time.Duration
	if planner.trip.TripEnd.Sub(currentTime).Hours() <= 12 {
		timeDelta = time.Duration(2) * time.Hour
	} else {
		timeDelta = time.Duration(4) * time.Hour
	}
	for !currentTime.After(planner.trip.TripEnd) {
		checkedTimes = append(checkedTimes, currentTime)
		currentTime = currentTime.Add(timeDelta)
	}
	durations = ants.NewTravelTimeMatrix(length, checkedTimes)
	distances = ants.NewDistanceMatrix(length, checkedTimes)
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
			DepartureTime: strconv.Itoa(int(t.Unix())),
			Mode:          planner.trip.TravelMode,
		}
		var resp *maps.DistanceMatrixResponse
		resp, err := planner.client.DistanceMatrix(context.Background(), r)
		if err != nil {
			return durations, distances, err
		}
		for i, row := range resp.Rows {
			for j, element := range row.Elements {
				if i != j {
					if element.Status == "OK" {
						if planner.trip.TravelMode == maps.TravelModeDriving {
							durations.Set(i, j, t, element.DurationInTraffic)
						} else {
							durations.Set(i, j, t, element.Duration)
						}
						distances.Set(i, j, t, int64(element.Distance.Meters))
					} else {
						return durations, distances, errors.New(fmt.Sprintf(
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

	return durations, distances, nil
}
