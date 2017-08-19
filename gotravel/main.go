package main

import (
	"log"

	"github.com/kr/pretty"
	//"golang.org/x/net/context"
	"googlemaps.github.io/maps"
	"github.com/mitchellh/mapstructure"
	gts "./gotravelsvc"
	"encoding/json"
)

const (
	API_KEY = "AIzaSyAT1X4AFtXRyBGOoE6ENqw5spvmUV28tSs"
)

func main() {
	_, err := maps.NewClient(maps.WithAPIKey(API_KEY))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	//var origin, destination string
	//{
	//	origin = "Warszawa, PL"
	//	destination = "Wrocław, PL"
	//}
	//{
	//	r := &maps.DistanceMatrixRequest{
	//		Origins:      []string{"Warszawa, PL", "Kraków, PL", "Wrocław, PL"},
	//		Destinations: []string{"Warszawa, PL", "Kraków, PL", "Wrocław, PL"},
	//		Mode:         maps.TravelModeDriving,
	//	}
	//	var resp *maps.DistanceMatrixResponse
	//	resp, err = c.DistanceMatrix(context.Background(), r)
	//	if err != nil {
	//		log.Fatalf("fatal error: %s", err)
	//	}
	//
	//	//pretty.Println(resp)
	//
	//	for i, from := range resp.OriginAddresses {
	//		row := resp.Rows[i]
	//		for j, to := range resp.DestinationAddresses {
	//			element := row.Elements[j]
	//			var (
	//				duration float64
	//				dist_m   int
	//				dist_hr  string
	//			)
	//			{
	//				duration = element.Duration.Seconds()
	//				dist_m = element.Distance.Meters
	//				dist_hr = element.Distance.HumanReadable
	//			}
	//
	//			fmt.Printf("%s -> %s\n"+
	//				"dur:\t%.2f s\n"+
	//				"dist:\t%s (%d m)\n\n",
	//				from, to,
	//				duration,
	//				dist_hr, dist_m,
	//			)
	//		}
	//	}
	//}

	//var place_id string

	//{
	//	r := &maps.GeocodingRequest{
	//		Address: "Bema Cafe, Wrocław",
	//	}
	//	var resp []maps.GeocodingResult
	//	resp, err = c.Geocode(context.Background(), r)
	//	if err != nil {
	//		log.Fatalf("fatal error: %s", err)
	//	}
	//
	//	pretty.Println(resp)
	//
	//	place_id = resp[0].PlaceID
	//}
	//
	//{
	//	r := &maps.PlaceDetailsRequest{
	//		PlaceID: place_id,
	//	}
	//	var resp maps.PlaceDetailsResult
	//	resp, err = c.PlaceDetails(context.Background(), r)
	//	if err != nil {
	//		log.Fatalf("fatal error: %s", err)
	//	}
	//
	//	pretty.Println(resp)
	//
	//}
	
	jsonRequest := []byte(`{
	"Mode": "address",
	"TripStart": [12, 0],
	"TripEnd": [20, 30],
	"TripDate": {
		"Day": 12,
		"Month": 6,
		"Year": 2017
	},
	"TravelModes": {
		"Driving": false,
		"Walking": true,
		"Transit": false,
		"Cycling": true
	},
	"Places": [
		{
			"Description": {
				"Name": "Bar Placuszek",
				"Street": "Jedności Narodowej",
				"Number": 12,
				"City": "Wrocław",
				"Country": "Poland"
			},
			"Priority": 5,
			"StayDuration": 45
		}
	]
}`)
	var tc gts.TripConfiguration
	
	err = json.Unmarshal(jsonRequest, &tc)
	
	if err != nil {
		println("error occurred", err.Error())
	} else {
		for _, place := range tc.Places {
			var ad gts.AddressDescription
			err = mapstructure.Decode(place.Description, &ad)
			if err != nil {
				println(err.Error())
			} else {
				place.Description = ad
			}
		}
		pretty.Println(tc)
	}
}
