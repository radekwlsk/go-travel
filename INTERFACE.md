# REQUEST

Requests can be made in three modes:
1. **Address**, where places are identified by their address match on Google Maps.
2. **Geo-coordinates**, where places are identified by their latitude and longitude on Google Maps.
3. **Name**, where places are search on Google Maps by provided name and 1st result is selected.

Requests are json queries:
```json
{
  "Mode": "address",
  "TripStart": ["12", "00"],
  "TripEnd": ["20", "30"],
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
      "Address": {
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
}
```
> `Priority` can be integer value in range 0-10, places with lower priority can be omitted to allow visiting more high-priority places.
>
> `StayDuration` is time that tourist plans to spend in place, will be used to calculate route optimizing for trip time and priorities. 