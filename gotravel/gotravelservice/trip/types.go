package trip

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/radekwlsk/go-travel/utils"
	"googlemaps.github.io/maps"
)

var ErrZeroResults = errors.New("google maps API query returned no result")

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
	Path          []int     `json:"path"`
	TravelMode    maps.Mode `json:"travelMode"`
}

func (t *Trip) CreateSchedule() {
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
	t.Schedule = strings.Join(sStrings, "\n")
}

type Configuration struct {
	APIKey              string         `json:"apiKey"`
	Mode                string         `json:"mode"`
	Language            string         `json:"language,omitempty"`
	TripStart           string         `json:"tripStart"`
	TripEnd             string         `json:"tripEnd"`
	TravelMode          string         `json:"travelMode,omitempty"`
	PlacesConfiguration []*PlaceConfig `json:"places"`
}

type PlaceDetails struct {
	PermanentlyClosed   bool                          `json:"closed"`
	OpeningHoursPeriods map[time.Weekday]OpeningHours `json:"openingHours"`
	Location            *time.Location                `json:"-"`
	FormattedAddress    string                        `json:"formattedAddress"`
	Name                string                        `json:"name"`
}

type OpeningHours struct {
	Open  string `json:"open"`
	Close string `json:"close"`
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
		offset := *resp.UTCOffset * 60
		name := strconv.Itoa(*resp.UTCOffset / 60)
		location = time.FixedZone(name, offset)
	}
	var openingHours = make(map[time.Weekday]OpeningHours, 7)

	if resp.OpeningHours != nil {
		for i := 0; i < 7; i++ {
			openingHours[time.Weekday(i)] = OpeningHours{}
		}

		for _, o := range resp.OpeningHours.Periods {
			if o.Open.Time == "" && o.Close.Time == "" {
				continue
			} else if o.Open.Time == "0000" && o.Close.Time == "" {
				for i := range openingHours {
					openingHours[i] = OpeningHours{
						Open:  "0000",
						Close: "2359",
					}
				}
				break
			} else {
				openingHours[o.Open.Day] = OpeningHours{
					Open:  o.Open.Time,
					Close: o.Close.Time,
				}
			}
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
	String() string
}

type AddressDescription struct {
	Name       string `json:"name,omitempty"`
	Street     string `json:"street"`
	Number     string `json:"number"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

func (ad *AddressDescription) IsEmpty() bool {
	return ad.Number == "" && ad.Street == "" && ad.City == ""
}

func (ad *AddressDescription) String() (address string) {
	address = fmt.Sprintf(
		"%s%s %s, %s%s",
		utils.IfThenElse(
			ad.Name == "",
			"",
			fmt.Sprintf("%s, ", ad.Name)),
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
		r := &maps.PlaceAutocompleteRequest{
			Input: ad.String(),
			Types: maps.AutocompletePlaceTypeEstablishment,
		}
		var resp maps.AutocompleteResponse
		resp, err := c.PlaceAutocomplete(context.Background(), r)
		if err != nil {
			if strings.Contains(err.Error(), "ZERO_RESULTS") {
				return "", ErrZeroResults
			}
			return "", err
		}
		placeId = resp.Predictions[0].PlaceID
	}

	return placeId, nil
}

type NameDescription struct {
	Name string `json:"name"`
}

func (nd *NameDescription) MapsPlaceID(service interface{}, c *maps.Client) (string, error) {
	var placeId string
	{
		r := &maps.PlaceAutocompleteRequest{
			Input: nd.Name,
			Types: maps.AutocompletePlaceTypeEstablishment,
		}
		var resp maps.AutocompleteResponse
		resp, err := c.PlaceAutocomplete(context.Background(), r)
		if err != nil {
			if strings.Contains(err.Error(), "ZERO_RESULTS") {
				return "", ErrZeroResults
			}
			return "", err
		}
		placeId = resp.Predictions[0].PlaceID
	}

	return placeId, nil
}

func (nd *NameDescription) String() string {
	return nd.Name
}

type PlaceIDDescription struct {
	PlaceID string `json:"placeId"`
}

func (pid *PlaceIDDescription) MapsPlaceID(service interface{}, c *maps.Client) (string, error) {
	return pid.PlaceID, nil
}

func (pid *PlaceIDDescription) String() string {
	return pid.PlaceID
}
