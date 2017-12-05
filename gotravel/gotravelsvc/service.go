package gotravelsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/planner"
	"github.com/afrometal/go-travel/gotravel/gotravelsvc/types"
	"github.com/google/uuid"
	"github.com/gregjones/httpcache"
	"github.com/mitchellh/mapstructure"
	"googlemaps.github.io/maps"
)

// Service interface definition and basic service methods implementation,
// the actual actions performed by service on data.

type Service interface {
	TripPlan(context.Context, types.TripConfiguration) (types.Trip, error)
}

type InmemService struct {
	tripConfigurationMap *sync.Map
	cacheTransport       *httpcache.Transport
}

func NewInmemService() Service {
	return &InmemService{
		tripConfigurationMap: &sync.Map{},
		cacheTransport:       httpcache.NewMemoryCacheTransport(),
	}
}

func (s *InmemService) TripPlan(ctx context.Context, tc types.TripConfiguration) (trip types.Trip, err error) {

	trip = types.Trip{
		Places:      make([]*types.TripPlace, len(tc.Places)),
		ClientID:    uuid.New(),
		TripStart:   tc.TripStart,
		TripEnd:     tc.TripEnd,
		TravelModes: tc.TravelModes.MapsModes(),
	}

	client, err := maps.NewClient(maps.WithAPIKey(tc.APIKey), maps.WithHTTPClient(s.cacheTransport.Client()))

	if err != nil {
		return trip, err
	}

	wg := sync.WaitGroup{}
	wg.Add(len(tc.Places))
	errChan := make(chan error, len(tc.Places))
	for i, p := range tc.Places {
		go func(i int, place *types.Place) {
			defer wg.Done()
			config := mapstructure.DecoderConfig{ErrorUnused: true}
			var placeID string
			switch tc.Mode {
			case "address":
				config.Result = &types.AddressDescription{}
			case "name":
				config.Result = &types.NameDescription{}
			case "id":
				config.Result = &types.PlaceIDDescription{}
			default:
				errChan <- fmt.Errorf("no such request mode: %s", tc.Mode)
				return
			}
			decoder, err := mapstructure.NewDecoder(&config)
			if err != nil {
				errChan <- err
				return
			}
			if err = decoder.Decode(place.Description); err != nil {
				errChan <- err
				return
			}
			if _, ok := config.Result.(types.Description); ok {
				place.Description = config.Result
			} else {
				errChan <- fmt.Errorf("could not parse Description")
				return
			}
			placeID, err = place.Description.(types.Description).GetPlaceID(s, client)
			if err != nil {
				errChan <- err
				return
			}
			trip.Places[i] = &types.TripPlace{Index: i, Place: place, PlaceID: placeID}
			err = trip.Places[i].SetDetails(s, client)
			if err != nil {
				errChan <- err
				return
			}
			errChan <- nil
		}(i, p)
	}
	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return trip, err
		}
	}

	var start, end bool

	for _, p := range trip.Places {
		if p.Place.Start {
			if start {
				return trip, errors.New("two start places defined")
			} else {
				start = true
				trip.StartPlace = p
			}
		}
		if p.Place.End {
			if end {
				return trip, errors.New("two end places defined")
			} else {
				end = true
				trip.EndPlace = p
			}
		}
	}

	s.tripConfigurationMap.Store(trip.ClientID, tc)

	p := planner.NewPlanner(client, &trip)
	trip.Steps, err = p.Evaluate()

	if err != nil {
		return trip, err
	}

	return trip, nil
}
