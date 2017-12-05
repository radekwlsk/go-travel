package types

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/afrometal/go-travel/utils"
	"github.com/google/uuid"
	"googlemaps.github.io/maps"
)

type Place struct {
	Priority     int         `json:"priority,omitempty"`
	StayDuration int         `json:"stayDuration,omitempty"`
	Description  interface{} `json:"description"`
	Start        bool        `json:"start,omitempty"`
	End          bool        `json:"end,omitempty"`
}

type TravelModes struct {
	Driving   bool `json:"driving"`
	Walking   bool `json:"walking"`
	Transit   bool `json:"transit"`
	Bicycling bool `json:"bicycling"`
}

func (tm *TravelModes) MapsModes() (modes []maps.Mode) {
	if tm.Driving {
		modes = append(modes, maps.TravelModeDriving)
	}
	if tm.Walking {
		modes = append(modes, maps.TravelModeWalking)
	}
	if tm.Transit {
		modes = append(modes, maps.TravelModeTransit)
	}
	if tm.Bicycling {
		modes = append(modes, maps.TravelModeBicycling)
	}
	return
}

type TripConfiguration struct {
	APIKey      string      `json:"apiKey"`
	Mode        string      `json:"mode"`
	TripStart   time.Time   `json:"tripStart"`
	TripEnd     time.Time   `json:"tripEnd"`
	TravelModes TravelModes `json:"travelModes"`
	Places      []*Place    `json:"places"`
}

type PlaceDetails struct {
	PermanentlyClosed   bool                      `json:"closed"`
	OpeningHoursPeriods []maps.OpeningHoursPeriod `json:"openingHours"`
	Location            *time.Location
	FormattedAddress    string
}

type TripPlace struct {
	Index     int          `json:"id"`
	Place     *Place       `json:"place"`
	PlaceID   string       `json:"placeId"`
	Arrival   time.Time    `json:"arrival,omitempty"`
	Departure time.Time    `json:"departure,omitempty"`
	Details   PlaceDetails `json:"details,omitempty"`
}

func (tp *TripPlace) SetDetails(service interface{}, c *maps.Client) error {
	r := &maps.PlaceDetailsRequest{
		PlaceID: tp.PlaceID,
	}
	var resp maps.PlaceDetailsResult
	resp, err := c.PlaceDetails(context.Background(), r)
	if err != nil {
		return err
	}
	var location *time.Location
	location = time.FixedZone(strconv.Itoa(resp.UTCOffset), resp.UTCOffset)
	tp.Details = PlaceDetails{
		PermanentlyClosed:   resp.PermanentlyClosed,
		OpeningHoursPeriods: resp.OpeningHours.Periods,
		Location:            location,
		FormattedAddress:    resp.FormattedAddress,
	}
	return nil
}

type Step struct {
	From       int           `json:"from"`
	To         int           `json:"to"`
	Duration   time.Duration `json:"time"`
	Distance   int64         `json:"distance"`
	TravelMode maps.Mode     `json:"mode"`
}

type Path struct {
	path  []int
	Steps []Step
	len   int
	loop  bool
	dummy bool
}

func NewPath(size int, loop bool) Path {
	return Path{make([]int, size), make([]Step, 0), size, loop, false}
}

func NewDummyPath() Path {
	return Path{dummy: true}
}

func (p *Path) Set(i, value int) {
	if i >= p.len {
		panic("array index out of bounds")
	}
	p.path[i] = value
}

func (p *Path) SetStep(i, to int, dur time.Duration, dist int64) {
	if i < 1 {
		panic("tried to set step to first place")
	}
	p.Set(i, to)
	from := p.At(i - 1)
	p.Steps = append(p.Steps, Step{
		From:     from,
		To:       to,
		Duration: dur,
		Distance: dist,
	})
}

func (p *Path) PathIndexes() []int {
	if p.loop {
		return append(p.path, p.path[0])
	} else {
		return p.path
	}
}

func (p *Path) Cut(i int) {
	p.path = p.path[:i]
	p.len = len(p.path)
}

func (p *Path) Append(value int) {
	p.path = append(p.path, value)
	p.len = len(p.path)
}

func (p *Path) At(i int) int {
	if i >= p.len {
		panic("Array index out of bounds")
	}
	return p.path[i]
}

func (p *Path) Size() int {
	return p.len
}

func (p *Path) Path() []int {
	return p.path
}

type Description interface {
	GetPlaceID(interface{}, *maps.Client) (string, error)
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

func (ad *AddressDescription) GetPlaceID(service interface{}, c *maps.Client) (string, error) {
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

func (nd *NameDescription) GetPlaceID(service interface{}, c *maps.Client) (string, error) {
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

func (pid *PlaceIDDescription) GetPlaceID(service interface{}, c *maps.Client) (string, error) {
	return pid.PlaceID, nil
}

type Trip struct {
	ClientID      uuid.UUID    `json:"clientId"`
	Places        []*TripPlace `json:"places"`
	StartPlace    *TripPlace   `json:"-"`
	EndPlace      *TripPlace   `json:"-"`
	TripStart     time.Time    `json:"tripStart"`
	TripEnd       time.Time    `json:"tripEnd"`
	TotalDistance int64        `json:"totalDistance"`
	Steps         []Step       `json:"steps"`
	TravelModes   []maps.Mode  `json:"-"`
}
