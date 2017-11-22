package gotravelsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/afrometal/go-travel/utils"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"googlemaps.github.io/maps"
)

// Service interface definition and basic service methods implementation,
// the actual actions performed by service on data.

type Service interface {
	TripPlan(context.Context, TripConfiguration) (Trip, error)
}

type JSON []byte

type TravelModes struct {
	Driving   bool `json:"driving"`
	Walking   bool `json:"walking"`
	Transit   bool `json:"transit"`
	Bicycling bool `json:"bicycling"`
}

type Description interface {
	getPlaceID(*inmemService, *maps.Client) (string, error)
}

type AddressDescription struct {
	Name       string `json:"name"`
	Street     string `json:"street"`
	Number     string `json:"number"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

func (ad *AddressDescription) toAddressString() (address string) {
	address = fmt.Sprintf(
		"%s, %s %s, %s%s",
		ad.Name,
		ad.Street,
		ad.Number,
		utils.IfThenElse(
			ad.PostalCode == "",
			ad.City,
			fmt.Sprintf("%s %s", ad.PostalCode, ad.City)),
		utils.IfThenElse(
			ad.Country == "",
			"",
			fmt.Sprintf(", %s", ad.Country)),
	)
	return
}

func (ad *AddressDescription) getPlaceID(service *inmemService, c *maps.Client) (string, error) {
	var placeId string
	{
		r := &maps.GeocodingRequest{
			Address: ad.toAddressString(),
		}
		var resp []maps.GeocodingResult
		resp, err := c.Geocode(context.Background(), r)
		if err != nil {
			return "", err
		}
		placeId = resp[0].PlaceID
	}

	return placeId, nil
}

type NameDescription struct {
	Name string `json:"name"`
}

func (nd *NameDescription) getPlaceID(service *inmemService, c *maps.Client) (string, error) {
	var placeId string
	{
		r := &maps.TextSearchRequest{
			Query: nd.Name,
		}
		var resp maps.PlacesSearchResponse
		resp, err := c.TextSearch(context.Background(), r)
		if err != nil {
			return "", err
		}
		placeId = resp.Results[0].PlaceID
	}

	return placeId, nil
}

type PlaceIDDescription struct {
	PlaceID string `json:"place_id"`
}

func (pid *PlaceIDDescription) getPlaceID(service *inmemService, c *maps.Client) (string, error) {
	return pid.PlaceID, nil
}

type Place struct {
	Priority     int         `json:"priority,omitempty"`
	StayDuration int         `json:"stayDuration,omitempty"`
	Description  interface{} `json:"description"`
	Start        bool        `json:"start,omitempty"`
	End          bool        `json:"end,omitempty"`
}

type TripConfiguration struct {
	APIKey      string      `json:"apiKey"`
	Mode        string      `json:"mode"`
	TripStart   time.Time   `json:"tripStart,omitempty"`
	TripEnd     time.Time   `json:"tripEnd,omitempty"`
	Timezone    string      `json:"timezone,omitempty"`
	TravelModes TravelModes `json:"travelModes"`
	Places      []*Place    `json:"places"`
}

type TripPlace struct {
	Index   int         `json:"id"`
	Place   *Place      `json:"place"`
	PlaceID string      `json:"placeId"`
	Arrival time.Time   `json:"arrival,omitempty"`
	Leave   time.Time   `json:"leave,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func (tp *TripPlace) setDetails(service *inmemService, c *maps.Client) error {
	r := &maps.PlaceDetailsRequest{
		PlaceID: tp.PlaceID,
	}
	var resp maps.PlaceDetailsResult
	resp, err := c.PlaceDetails(context.Background(), r)
	if err != nil {
		return err
	}
	tp.Details = resp.OpeningHours
	return nil
}

type Trip struct {
	ClientID   uuid.UUID    `json:"clientId"`
	Places     []*TripPlace `json:"places"`
	StartPlace *TripPlace   `json:"startPlace"`
	EndPlace   *TripPlace   `json:"endPlace"`
	TripStart  time.Time    `json:"tripStart"`
	TripEnd    time.Time    `json:"tripEnd"`
}

func (t *Trip) Evaluate(c *maps.Client) error {
	p := NewPlanner(c, t)
	return p.Evaluate()
}

type inmemService struct {
	tripConfigurationMap *sync.Map
}

func NewInmemService() Service {
	return &inmemService{
		tripConfigurationMap: &sync.Map{},
	}
}

func (s *inmemService) TripPlan(ctx context.Context, tc TripConfiguration) (trip Trip, err error) {

	trip = Trip{
		Places:    make([]*TripPlace, len(tc.Places)),
		ClientID:  uuid.New(),
		TripStart: tc.TripStart,
		TripEnd:   tc.TripEnd,
	}

	client, err := maps.NewClient(maps.WithAPIKey(tc.APIKey))
	if err != nil {
		return trip, err
	}

	wg := sync.WaitGroup{}
	wg.Add(len(tc.Places))
	errChan := make(chan error, len(tc.Places))
	for i, p := range tc.Places {
		go func(i int, place *Place) {
			defer wg.Done()
			config := mapstructure.DecoderConfig{ErrorUnused: true}
			var placeID string
			switch tc.Mode {
			case "address":
				config.Result = &AddressDescription{}
			case "name":
				config.Result = &NameDescription{}
			case "id":
				config.Result = &PlaceIDDescription{}
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
			if _, ok := config.Result.(Description); ok {
				place.Description = config.Result
			} else {
				errChan <- fmt.Errorf("could not parse Description")
				return
			}
			placeID, err = place.Description.(Description).getPlaceID(s, client)
			if err != nil {
				errChan <- err
				return
			}
			trip.Places[i] = &TripPlace{Index: i, Place: place, PlaceID: placeID}
			err = trip.Places[i].setDetails(s, client)
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
				return trip, errors.New("Two start places defined.")
			} else {
				start = true
				trip.StartPlace = p
			}
		}
		if p.Place.End {
			if end {
				return trip, errors.New("Two end places defined.")
			} else {
				end = true
				trip.EndPlace = p
			}
		}
	}

	s.tripConfigurationMap.Store(trip.ClientID, tc)

	err = trip.Evaluate(client)

	if err != nil {
		return trip, err
	}

	return trip, nil
}
