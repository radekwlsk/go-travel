package gotravelsvc

import (
	"context"
	"fmt"
	"sync"

	"github.com/AfroMetal/go-travel/utils"
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
	PostalCode string `json:"postalcode,omitempty"`
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
	var place_id string
	{
		r := &maps.GeocodingRequest{
			Address: ad.toAddressString(),
		}
		var resp []maps.GeocodingResult
		resp, err := c.Geocode(context.Background(), r)
		if err != nil {
			return "", err
		}
		place_id = resp[0].PlaceID
	}

	return place_id, nil
}

type NameDescription struct {
	Name string `json:"name"`
}

func (nd *NameDescription) getPlaceID(service *inmemService, c *maps.Client) (string, error) {
	var place_id string
	{
		r := &maps.TextSearchRequest{
			Query: nd.Name,
		}
		var resp maps.PlacesSearchResponse
		resp, err := c.TextSearch(context.Background(), r)
		if err != nil {
			return "", err
		}
		place_id = resp.Results[0].PlaceID
	}

	return place_id, nil
}

type PlaceIDDescription struct {
	PlaceID string `json:"place_id"`
}

func (pid *PlaceIDDescription) getPlaceID(service *inmemService, c *maps.Client) (string, error) {
	return pid.PlaceID, nil
}

type Date struct {
	Day   int `json:"d"`
	Month int `json:"m"`
	Year  int `json:"y"`
}

type Place struct {
	Priority     int         `json:"priority,omitempty"`
	StayDuration int         `json:"stay_duration,omitempty"`
	Description  interface{} `json:"description"`
}

type TripConfiguration struct {
	APIKey      string      `json:"api_key"`
	Mode        string      `json:"mode"`
	TripStart   [2]int      `json:"trip_start,omitempty"`
	TripEnd     [2]int      `json:"trip_end,omitempty"`
	TripDate    Date        `json:"trip_date"`
	TravelModes TravelModes `json:"travel_modes"`
	Places      []*Place    `json:"places"`
}

type TripPlace struct {
	Index   int    `json:"id"`
	Place   *Place `json:"place"`
	PlaceID string `json:"place_id"`
}

type Trip struct {
	ClientID uuid.UUID    `json:"client_id"`
	Places   []*TripPlace `json:"places"`
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
		Places:   make([]*TripPlace, len(tc.Places)),
		ClientID: uuid.New(),
	}

	client, err := maps.NewClient(maps.WithAPIKey(tc.APIKey))
	if err != nil {
		return Trip{}, err
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
			place.Description = config.Result
			placeID, err = place.Description.(Description).getPlaceID(s, client)
			if err != nil {
				errChan <- err
				return
			}
			trip.Places[i] = &TripPlace{Index: i, Place: place, PlaceID: placeID}
			errChan <- nil
		}(i, p)
	}
	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return Trip{}, err
		}
	}

	s.tripConfigurationMap.Store(trip.ClientID, tc)

	err = trip.Evaluate(client)
	if err != nil {
		return Trip{}, err
	}

	return
}
