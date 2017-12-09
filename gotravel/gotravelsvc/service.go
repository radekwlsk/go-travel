package gotravelsvc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/planner"
	"github.com/afrometal/go-travel/gotravel/gotravelsvc/types"
	"github.com/afrometal/go-travel/utils"
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

var (
	ErrAPIKeyEmpty = errors.New("request must contain Google Maps API Key as 'apiKey'")

	ErrModeEmpty = errors.New("request places description mode must be provided as 'mode'")

	ErrTripStartEmpty = errors.New("request must contain trip start time in 'YYYY-MM-DDThh:mm:ssZ' format as" +
		" 'tripStart'")

	ErrTripEndEmpty = errors.New("request must contain trip end time in 'YYYY-MM-DDThh:mm:ssZ' format as" +
		" 'tripEnd'")

	ErrNotEnoughPlaces = errors.New("request must contain at least two places")

	ErrBadTimeFormat = errors.New("tripStart/tripEnd time must be provided in 'YYYY-MM-DDThh:mm:ssZ' format")

	ErrBadTime = errors.New("tripStart/tripEnd time can not be in the past")

	ErrEndBeforeStart = errors.New("tripEnd time is before tripStart time")

	ErrBadDescription = errors.New("could not parse place description")

	ErrTwoStartPlaces = errors.New("more than one place marked as start")

	ErrTwoEndPlaces = errors.New("more than one place marked as end")

	ErrBadMode = errors.New(fmt.Sprintf("place description mode is not valid, available modes are: %s",
		strings.Join(types.ModeOptions, ", ")))

	ErrBadTravelMode = errors.New(fmt.Sprintf(
		"travelMode is not a valid, available modes are: %s",
		strings.Join(types.TravelModeOptions, ", ")))
)

type service struct {
	cacheTransport *httpcache.Transport
}

func NewService() Service {
	return &service{
		cacheTransport: httpcache.NewMemoryCacheTransport(),
	}
}

func (s *service) TripPlan(ctx context.Context, tc types.TripConfiguration) (trip types.Trip, err error) {

	if tc.APIKey == "" {
		return types.Trip{}, ErrAPIKeyEmpty
	}

	if tc.Mode == "" {
		return types.Trip{}, ErrModeEmpty
	} else if !utils.StringIn(tc.Mode, types.ModeOptions) {
		return types.Trip{}, ErrBadMode
	}

	if tc.TravelMode == "" {
		tc.TravelMode = types.TravelModeOptions[0]
	} else if !utils.StringIn(tc.TravelMode, types.TravelModeOptions) {
		return types.Trip{}, ErrBadTravelMode
	}

	var ts, te time.Time
	var now = time.Now()

	if tc.TripStart == "" {
		return types.Trip{}, ErrTripStartEmpty
	} else if ts, err = time.Parse(time.RFC3339, tc.TripStart); err != nil {
		return types.Trip{}, ErrBadTimeFormat
	} else if ts.Before(now) {
		return types.Trip{}, ErrBadTime
	}

	if tc.TripEnd == "" {
		return types.Trip{}, ErrTripEndEmpty
	} else if te, err = time.Parse(time.RFC3339, tc.TripEnd); err != nil {
		return types.Trip{}, ErrBadTimeFormat
	} else if te.Before(now) {
		return types.Trip{}, ErrBadTime
	}

	if te.Before(ts) {
		return types.Trip{}, ErrEndBeforeStart
	}

	var pLen int

	if pLen = len(tc.Places); pLen < 2 {
		return types.Trip{}, ErrNotEnoughPlaces
	}

	trip = types.Trip{
		Places:     make([]*types.TripPlace, pLen),
		ClientID:   uuid.New(),
		TripStart:  ts,
		TripEnd:    te,
		TravelMode: maps.Mode(tc.TravelMode),
	}

	client, err := maps.NewClient(maps.WithAPIKey(tc.APIKey), maps.WithHTTPClient(s.cacheTransport.Client()))

	if err != nil {
		println("error 1")
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
				errChan <- ErrBadMode
				return
			}
			decoder, err := mapstructure.NewDecoder(&config)
			if err != nil {
				println("error 2")
				errChan <- err
				return
			}
			if err = decoder.Decode(place.Description); err != nil {
				errChan <- ErrBadDescription
				return
			}
			if _, ok := config.Result.(types.Description); ok {
				place.Description = config.Result
			} else {
				errChan <- ErrBadDescription
				return
			}
			switch tc.Mode {
			case "address":
				if place.Description.(*types.AddressDescription).IsEmpty() {
					errChan <- ErrBadDescription
					return
				}
			case "name":
				if place.Description.(*types.NameDescription).Name == "" {
					errChan <- ErrBadDescription
					return
				}
			case "id":
				if place.Description.(*types.PlaceIDDescription).PlaceID == "" {
					errChan <- ErrBadDescription
					return
				}
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
				return trip, ErrTwoEndPlaces
			} else {
				start = true
				trip.StartPlace = p
			}
		}
		if p.Place.End {
			if end {
				return trip, ErrTwoEndPlaces
			} else {
				end = true
				trip.EndPlace = p
			}
		}
	}

	p := planner.NewPlanner(client, &trip)
	trip.Steps, err = p.Evaluate()

	if err != nil {
		return trip, err
	}

	return trip, nil
}
