package trip

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/afrometal/go-travel/utils"
	"googlemaps.github.io/maps"
)

type PlaceConfig struct {
	Priority     int         `json:"priority,omitempty"`
	StayDuration int         `json:"stayDuration,omitempty"`
	Description  interface{} `json:"description"`
	Start        bool        `json:"start,omitempty"`
	End          bool        `json:"end,omitempty"`
}

var TravelModeOptions = []string{
	"walking",
	"bicycling",
	"transit",
	"driving",
}

var ModeOptions = []string{
	"name",
	"address",
	"id",
}

type Trip struct {
	Places        []*Place  `json:"places"`
	StartPlace    *Place    `json:"-"`
	EndPlace      *Place    `json:"-"`
	TripStart     time.Time `json:"tripStart"`
	TripEnd       time.Time `json:"tripEnd"`
	TotalDistance int64     `json:"totalDistance"`
	Steps         []Step    `json:"steps"`
	Schedule      string    `json:"schedule"`
	TravelMode    maps.Mode `json:"travelMode"`
}

func (t *Trip) CreateSchedule() string {
	var dStrings = make([]string, len(t.Places))
	var aStrings = make([]string, len(t.Places))

	for i, p := range t.Places {
		aStrings[i] = p.Arrival.Format("Mon Jan 2, 15:04")
		if p.Departure.Day() != p.Arrival.Day() {
			dStrings[i] = p.Departure.Format("Mon Jan 2, 15:04")
		} else {
			dStrings[i] = p.Departure.Format("15:04")
		}
	}

	var sStrings = make([]string, len(t.Steps))

	for i, s := range t.Steps {
		sStrings[i] = fmt.Sprintf(
			"[%s - %s] %s, %s",
			aStrings[s.From],
			dStrings[s.From],
			t.Places[s.From].Details.Name,
			t.Places[s.From].Details.FormattedAddress)
	}

	if t.StartPlace != nil && t.EndPlace != t.StartPlace {
		last := t.Steps[len(t.Steps)-1].To
		sStrings = append(sStrings, fmt.Sprintf(
			"[%s - %s] %s, %s",
			aStrings[last],
			dStrings[last],
			t.Places[last].Details.Name,
			t.Places[last].Details.FormattedAddress))

	} else if t.EndPlace != nil {
		sStrings = append(sStrings, fmt.Sprintf(
			"[%s] %s, %s",
			t.TripEnd.Format("Mon Jan 2, 15:04"),
			t.Places[t.EndPlace.Index].Details.Name,
			t.Places[t.EndPlace.Index].Details.FormattedAddress))
	}
	return strings.Join(sStrings, "\n")
}

type Configuration struct {
	APIKey              string         `json:"apiKey"`
	Mode                string         `json:"mode"`
	Language            string         `json:"language"`
	TripStart           string         `json:"tripStart"`
	TripEnd             string         `json:"tripEnd"`
	TravelMode          string         `json:"travelMode,omitempty"`
	PlacesConfiguration []*PlaceConfig `json:"places"`
}

type PlaceDetails struct {
	PermanentlyClosed   bool                      `json:"closed"`
	OpeningHoursPeriods []maps.OpeningHoursPeriod `json:"openingHours"`
	Location            *time.Location            `json:"-"`
	FormattedAddress    string                    `json:"formattedAddress"`
	Name                string                    `json:"name"`
}

type Place struct {
	Index        int          `json:"id"`
	StayDuration int          `json:"stayDuration"`
	Priority     int          `json:"priority"`
	PlaceID      string       `json:"-"`
	Arrival      time.Time    `json:"arrival,omitempty"`
	Departure    time.Time    `json:"departure,omitempty"`
	Details      PlaceDetails `json:"details,omitempty"`
}

func (p *Place) SetDetails(service interface{}, c *maps.Client, lang string) error {
	r := &maps.PlaceDetailsRequest{
		PlaceID:  p.PlaceID,
		Language: lang,
	}
	var resp maps.PlaceDetailsResult
	resp, err := c.PlaceDetails(context.Background(), r)
	if err != nil {
		return err
	}
	var location *time.Location
	{
		offset := resp.UTCOffset * 60
		name := strconv.Itoa(resp.UTCOffset / 60)
		location = time.FixedZone(name, offset)
	}
	var openingHours = make([]maps.OpeningHoursPeriod, 7)
	for _, o := range resp.OpeningHours.Periods {
		openingHours[o.Open.Day] = o
	}

	for i, o := range openingHours {
		if o.Open.Time == "" || o.Close.Time == "" {
			o.Open.Day = time.Weekday(i)
			o.Close.Day = time.Weekday(i)
		}
	}

	p.Details = PlaceDetails{
		PermanentlyClosed:   resp.PermanentlyClosed,
		OpeningHoursPeriods: openingHours,
		Location:            location,
		FormattedAddress:    resp.FormattedAddress,
		Name:                resp.Name,
	}
	return nil
}

type Step struct {
	From     int           `json:"from"`
	To       int           `json:"to"`
	Duration time.Duration `json:"time"`
	Distance int64         `json:"distance"`
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
	if to != p.path[0] {
		p.Set(i, to)
	}
	from := p.At(i - 1)
	p.Steps = append(p.Steps, Step{
		From:     from,
		To:       to,
		Duration: dur / time.Minute,
		Distance: dist,
	})
}

func (p *Path) Cut(i int) {
	p.path = p.path[:i]
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
	MapsPlaceID(interface{}, *maps.Client) (string, error)
}

type AddressDescription struct {
	Name       string `json:"name"`
	Street     string `json:"street"`
	Number     string `json:"number"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

func (ad *AddressDescription) IsEmpty() bool {
	return ad.Name == "" && ad.Street == "" && ad.City == ""
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

func (ad *AddressDescription) MapsPlaceID(service interface{}, c *maps.Client) (string, error) {
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

func (nd *NameDescription) MapsPlaceID(service interface{}, c *maps.Client) (string, error) {
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

func (pid *PlaceIDDescription) MapsPlaceID(service interface{}, c *maps.Client) (string, error) {
	return pid.PlaceID, nil
}
