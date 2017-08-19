package gotravelsvc

import (
	"context"
	"sync"
	"github.com/kr/pretty"
	"log"
	"github.com/mitchellh/mapstructure"
	"fmt"
)

// Service interface definition and basic service methods implementation,
// the actual actions performed by service on data.

type Service interface {
	TripPlan(context.Context, TripConfiguration) (string, error)
}

type JSON []byte

type TravelModes struct {
	Driving bool `json:"Driving"`
	Walking bool `json:"Walking"`
	Transit bool `json:"Transit"`
	Cycling bool `json:"Cycling"`
}

type AddressDescription struct {
	Name    string `json:"Name"`
	Street  string `json:"Street"`
	Number  int    `json:"Number"`
	City    string `json:"City"`
	Country string `json:"Country"`
}

type GeoCoordinatesDescription struct {
	Lat string `json:"Lat"`
	Lng string `json:"Lng"`
}

type NameDescription struct {
	Name string `json:"Name"`
}

type PlaceIDDescription struct {
	PlaceID string `json:"PlaceID"`
}

type Date struct {
	Day   int `json:"Day"`
	Month int `json:"Month"`
	Year  int `json:"Year"`
}

type Place struct {
	Priority     int         `json:"Priority,omitempty"`
	StayDuration int         `json:"StayDuration,omitempty"`
	Description  interface{} `json:"Description"`
}

type TripConfiguration struct {
	APIKey      string      `json:"APIKey"`
	Mode        string      `json:"Mode"`
	TripStart   [2]int      `json:"TripStart,omitempty"`
	TripEnd     [2]int      `json:"TripEnd,omitempty"`
	TripDate    Date        `json:"TripDate"`
	TravelModes TravelModes `json:"TravelModes"`
	Places      []*Place    `json:"Places"`
}

//type Trip struct {
//
//}

type inmemService struct {
	mtx sync.RWMutex
	m   map[string]TripConfiguration
}

func NewInmemService() Service {
	return &inmemService{
		m: map[string]TripConfiguration{},
	}
}

func (s *inmemService) TripPlan(ctx context.Context, tc TripConfiguration) (string, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	
	config := mapstructure.DecoderConfig{ErrorUnused: true, }
	for _, place := range tc.Places {
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
			return "", fmt.Errorf("No such Mode: %s", tc.Mode)
		}
		decoder, err := mapstructure.NewDecoder(&config)
		if err != nil {
			return "", err
		}
		if err = decoder.Decode(place.Description); err != nil {
			return "", err
		}
		place.Description = config.Result
	}
	
	s.m[tc.APIKey] = tc
	
	decoded := pretty.Sprint(tc)
	log.Printf("Decoded request to:\n%s", decoded)
	
	return decoded, nil
}
