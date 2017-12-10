package gotravelsvc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/planner"
	"github.com/afrometal/go-travel/gotravel/gotravelsvc/trip"
	"github.com/afrometal/go-travel/utils"
	"github.com/gregjones/httpcache"
	"github.com/mitchellh/mapstructure"
	"googlemaps.github.io/maps"
)

// Service interface definition and basic service methods implementation,
// the actual actions performed by service on data.
type Service interface {
	TripPlan(context.Context, trip.Configuration) (trip.Trip, error)
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
		strings.Join(trip.ModeOptions, ", ")))

	ErrBadTravelMode = errors.New(fmt.Sprintf(
		"travelMode is not a valid, available modes are: %s",
		strings.Join(trip.TravelModeOptions, ", ")))
)

type service struct {
	cacheTransport *httpcache.Transport
}

func NewService() Service {
	return &service{
		cacheTransport: httpcache.NewMemoryCacheTransport(),
	}
}

func (s *service) TripPlan(ctx context.Context, tc trip.Configuration) (t trip.Trip, err error) {

	if tc.APIKey == "" {
		return trip.Trip{}, ErrAPIKeyEmpty
	}

	if tc.Mode == "" {
		return trip.Trip{}, ErrModeEmpty
	} else if !utils.StringIn(tc.Mode, trip.ModeOptions) {
		return trip.Trip{}, ErrBadMode
	}

	if tc.TravelMode == "" {
		tc.TravelMode = trip.TravelModeOptions[0]
	} else if !utils.StringIn(tc.TravelMode, trip.TravelModeOptions) {
		return trip.Trip{}, ErrBadTravelMode
	}

	var ts, te time.Time
	var now = time.Now()

	if tc.TripStart == "" {
		return trip.Trip{}, ErrTripStartEmpty
	} else if ts, err = time.Parse(time.RFC3339, tc.TripStart); err != nil {
		return trip.Trip{}, ErrBadTimeFormat
	} else if ts.Before(now) {
		return trip.Trip{}, ErrBadTime
	}

	if tc.TripEnd == "" {
		return trip.Trip{}, ErrTripEndEmpty
	} else if te, err = time.Parse(time.RFC3339, tc.TripEnd); err != nil {
		return trip.Trip{}, ErrBadTimeFormat
	} else if te.Before(now) {
		return trip.Trip{}, ErrBadTime
	}

	if te.Before(ts) {
		return trip.Trip{}, ErrEndBeforeStart
	}

	var pLen int

	if pLen = len(tc.PlacesConfiguration); pLen < 2 {
		return trip.Trip{}, ErrNotEnoughPlaces
	}

	t = trip.Trip{
		Places:     make([]*trip.Place, pLen),
		TripStart:  ts,
		TripEnd:    te,
		TravelMode: maps.Mode(tc.TravelMode),
	}

	c, err := maps.NewClient(maps.WithAPIKey(tc.APIKey), maps.WithHTTPClient(s.cacheTransport.Client()))

	if err != nil {
		println("error 1")
		return t, err
	}

	wg := sync.WaitGroup{}
	wg.Add(len(tc.PlacesConfiguration))
	errChan := make(chan error, len(tc.PlacesConfiguration))
	for i, p := range tc.PlacesConfiguration {
		go func(i int, place *trip.PlaceConfig) {
			defer wg.Done()
			config := mapstructure.DecoderConfig{ErrorUnused: true}
			var placeID string
			switch tc.Mode {
			case "address":
				config.Result = &trip.AddressDescription{}
			case "name":
				config.Result = &trip.NameDescription{}
			case "id":
				config.Result = &trip.PlaceIDDescription{}
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
			if _, ok := config.Result.(trip.Description); ok {
				place.Description = config.Result
			} else {
				errChan <- ErrBadDescription
				return
			}
			switch tc.Mode {
			case "address":
				if place.Description.(*trip.AddressDescription).IsEmpty() {
					errChan <- ErrBadDescription
					return
				}
			case "name":
				if place.Description.(*trip.NameDescription).Name == "" {
					errChan <- ErrBadDescription
					return
				}
			case "id":
				if place.Description.(*trip.PlaceIDDescription).PlaceID == "" {
					errChan <- ErrBadDescription
					return
				}
			}
			placeID, err = place.Description.(trip.Description).MapsPlaceID(s, c)
			if err != nil {
				errChan <- err
				return
			}
			t.Places[i] = &trip.Place{
				Index:        i,
				StayDuration: place.StayDuration,
				Priority:     place.Priority,
				PlaceID:      placeID,
			}
			if place.Start {
				if t.StartPlace != nil {
					errChan <- ErrTwoStartPlaces
					return
				}
				t.StartPlace = t.Places[i]
			}
			if place.End {
				if t.EndPlace != nil {
					errChan <- ErrTwoEndPlaces
					return
				}
				t.EndPlace = t.Places[i]
			}
			err = t.Places[i].SetDetails(s, c, tc.Language)
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
			return t, err
		}
	}

	p := planner.NewPlanner(c, &t)
	t.Steps, err = p.Evaluate()

	if err != nil {
		return t, err
	}

	t.Schedule = t.CreateSchedule()

	return t, nil
}
