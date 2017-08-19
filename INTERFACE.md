# REQUEST

Requests can be made in four modes:
1. **Address**, where places are identified by their address match on Google Maps.
2. **Geo-coordinates**, where places are identified by their latitude and longitude on Google Maps.
3. **Name**, where places are searched on Google Maps by provided name and 1st result is selected.
4. **PlaceID**, where places are identified by their Google Maps API PlaceID

```
{
  "APIKey" : string,
  "Mode": ["address"|"geo"|"name"|"id"],
  "TripStart": [int 0-23, int 0-59],
  "TripEnd": [int 0-23, int 0-59],
  "TripDate": {
    "Day": int 1-31,
    "Month": int 1-12,
    "Year": int year
  },
  "TravelModes": {
    "Driving": [true|false],
    "Walking": [true|false],
    "Transit": [true|false],
    "Cycling": [true|false]
  },
  "Places": [
    {
      "Description": {},
      "Priority": int 1-10,
      "StayDuration": int minutes
    }
  ]
}
```
> `Description` mode specific place description used to identify specific location:
> - in **Address** mode:
>   ```json
>   {
>     "Name": "Bar Placuszek",
>     "Street": "Jedności Narodowej",
>     "Number": 12,
>     "City": "Wrocław", 
>     "Country": "Poland"
>   }
>   ```
>
> - in **Geo-coordinates** mode:
>   ```json
>   {
>     "Lat": 12.21341,
>     "Lng": -43.21342
>   }
>   ```
>
> - in **Name** mode:
>   ```json
>   {
>     "Name": "Bar Placuszek"
>   }
>   ```
>
> - in **PlaceID** mode:
>   ```json
>   {
>     "PlaceID": "AF346Q#ABTG&EASF1!@"
>   }
>   ```
>
> `Priority` can be integer value in range 0-10, places with lower priority can be omitted to allow visiting more high-priority places.
>
> `StayDuration` is time that tourist plans to spend in place, will be used to calculate route optimizing for trip time and priorities. 

# RESPONSE
