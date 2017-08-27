package gotravelsvc

import (
	"context"
	"fmt"
	"log"
	"sync"

	"errors"
	"github.com/AfroMetal/go-travel/utils"
	"github.com/google/uuid"
	"github.com/kr/pretty"
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
	getPlaceID(*inmemService, string) (string, error)
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

func (ad *AddressDescription) getPlaceID(service *inmemService, apiKey string) (string, error) {
	c, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return "", err
	}

	var place_id string

	{
		r := &maps.GeocodingRequest{
			Address: ad.toAddressString(),
		}
		var resp []maps.GeocodingResult
		resp, err = c.Geocode(context.Background(), r)
		if err != nil {
			return "", err
		}
		place_id = resp[0].PlaceID
	}

	return place_id, nil
}

type GeoCoordinatesDescription maps.LatLng

func (gcd *GeoCoordinatesDescription) getPlaceID(*inmemService, string) (string, error) {
	return "", errors.New("Not yet implemented")
}

type NameDescription struct {
	Name string `json:"name"`
}

func (nd *NameDescription) getPlaceID(*inmemService, string) (string, error) {
	return "", errors.New("Not yet implemented")
}

type PlaceIDDescription struct {
	PlaceID string `json:"place_id"`
}

func (pid *PlaceIDDescription) getPlaceID(*inmemService, string) (string, error) {
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

func (trip *Trip) Evaluate() error {
	return nil
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
	config := mapstructure.DecoderConfig{ErrorUnused: true}
	var placeID string
	for i, place := range tc.Places {
		switch tc.Mode {
		case "address":
			config.Result = &AddressDescription{}
		case "geo":
			config.Result = &GeoCoordinatesDescription{}
		case "name":
			config.Result = &NameDescription{}
		case "id":
			config.Result = &PlaceIDDescription{}
		default:
			return Trip{}, fmt.Errorf("No such request mode: %s", tc.Mode)
		}
		decoder, err := mapstructure.NewDecoder(&config)
		if err != nil {
			return Trip{}, err
		}
		if err = decoder.Decode(place.Description); err != nil {
			return Trip{}, err
		}
		place.Description = config.Result
		placeID, err = place.Description.(Description).getPlaceID(s, tc.APIKey)
		if err != nil {
			return Trip{}, err
		}
		trip.Places[i] = &TripPlace{Index: i, Place: place, PlaceID: placeID}
	}

	s.tripConfigurationMap.Store(trip.ClientID, tc)

	err = trip.Evaluate()
	if err != nil {
		return Trip{}, err
	}

	decoded := pretty.Sprint(tc)
	log.Printf("Decoded request to:\n%s", decoded)

	return
}
