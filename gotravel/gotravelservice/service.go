package gotravelservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gregjones/httpcache"
	"github.com/mitchellh/mapstructure"
	"github.com/radekwlsk/go-travel/gotravel/gotravelservice/planner"
	"github.com/radekwlsk/go-travel/gotravel/gotravelservice/trip"
	"github.com/radekwlsk/go-travel/utils"
	"googlemaps.github.io/maps"
)

// Service interface definition and basic service methods implementation,
// the actual actions performed by service on data.
type Service interface {
	TripPlan(context.Context, trip.Configuration) (trip.Trip, error)
}

func New(logger log.Logger) Service {
	var s Service
	{
		s = NewService()
		s = NewLoggingMiddleware(log.With(logger, "layer", "service"))(s)
	}
	return s
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

	ErrTwoStartPlaces = errors.New("more than one place marked as start")

	ErrTwoEndPlaces = errors.New("more than one place marked as end")

	ErrBadMode = errors.New(fmt.Sprintf("place description mode is not valid, available modes are: %s",
		strings.Join(trip.ModeOptions, ", ")))

	ErrBadTravelMode = errors.New(fmt.Sprintf(
		"travelMode is not a valid, available modes are: %s",
		strings.Join(trip.TravelModeOptions, ", ")))
)

type ErrBadDescription struct {
	Place *trip.PlaceConfig
}

func (err ErrBadDescription) Error() string {
	return fmt.Sprintf("could not parse place description of %s", err.Place.Description)
}

type ErrDescriptionInaccurate struct {
	Place *trip.PlaceConfig
}

func (err ErrDescriptionInaccurate) Error() string {
	return fmt.Sprintf("description not accurate, no results found for %s", err.Place.Description)
}

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
				errChan <- err
				return
			}
			if err = decoder.Decode(place.Description); err != nil {
				errChan <- ErrBadDescription{place}
				return
			}
			if _, ok := config.Result.(trip.Description); ok {
				place.Description = config.Result
			} else {
				errChan <- ErrBadDescription{place}
				return
			}
			switch tc.Mode {
			case "address":
				if place.Description.(*trip.AddressDescription).IsEmpty() {
					errChan <- ErrBadDescription{place}
					return
				}
			case "name":
				if place.Description.(*trip.NameDescription).Name == "" {
					errChan <- ErrBadDescription{place}
					return
				}
			case "id":
				if place.Description.(*trip.PlaceIDDescription).PlaceID == "" {
					errChan <- ErrBadDescription{place}
					return
				}
			}
			placeID, err = place.Description.(trip.Description).MapsPlaceID(s, c)
			switch err {
			case nil:
				break
			case trip.ErrZeroResults:
				errChan <- ErrDescriptionInaccurate{place}
				return
			default:
				errChan <- err
				return
			}
			t.Places[i] = &trip.Place{
				Index:        i,
				StayDuration: place.StayDuration,
				Priority:     place.Priority,
				PlaceID:      placeID,
			}
			if t.Places[i].Priority > 10 {
				t.Places[i].Priority = 10
			} else if t.Places[i].Priority < 0 {
				t.Places[i].Priority = 0
			}
			if t.Places[i].StayDuration < 0 {
				t.Places[i].StayDuration = 0
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

	{
		tswd := t.TripStart.Weekday()
		h, m, _ := t.TripStart.Clock()
		fh, fm := 23, 59

		for _, p := range t.Places {
			var o string
			for wd, op := range p.Details.OpeningHoursPeriods {
				if wd == tswd {
					o = op.Open
					break
				}
			}
			if o != "" {
				oh, _ := strconv.Atoi(o[:2])
				om, _ := strconv.Atoi(o[2:])
				if oh < fh || om < fm {
					fh, fm = oh, om
				}
			}
		}

		if fh > h {
			t.TripStart = t.TripStart.Add(time.Duration(fh-h) * time.Hour)
			t.TripStart = t.TripStart.Add(time.Duration(fm-m) * time.Minute)
		}
	}

	p := planner.NewPlanner(c, &t)
	err = p.Evaluate()

	if err != nil {
		return t, err
	}

	return t, nil
}
