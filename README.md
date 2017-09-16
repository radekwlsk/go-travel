# REQUESTS

## Format

Requests can be made in four modes:
1. **Address**, where places are identified by their address match on Google Maps.
3. **Name**, where places are searched on Google Maps by provided name and 1st result is selected.
4. **PlaceID**, where places are identified by their Google Maps API PlaceID

```
{
  "api_key" : string,
  "mode": ["address"|"name"|"id"],
  "trip_start": [int 0-23, int 0-59],
  "trip_end": [int 0-23, int 0-59],
  "trip_date": {
    "d": int 1-31,
    "m": int 1-12,
    "y": int year
  },
  "travel_modes": {
    "driving": [true|false],
    "walking": [true|false],
    "transit": [true|false],
    "bicycling": [true|false]
  },
  "places": [
    {
      "description": {},
      "priority": int 1-10,
      "stay_duration": int minutes
    }
  ]
}
```
> `Description` mode specific place description used to identify specific location:
> - in **Address** mode:
>   ```json
>   {
>     "name": "Bar Placuszek",
>     "street": "Jedności Narodowej",
>     "number": "12",
>     "city": "Wrocław",
>     "postal_code": "50-309",
>     "country": "Poland"
>   }
>   ```
>
> - in **Name** mode:
>   ```json
>   {
>     "name": "Bar Placuszek"
>   }
>   ```
>
> - in **PlaceID** mode:
>   ```json
>   {
>     "place_id": "AF346Q#ABTG&EASF1!@"
>   }
>   ```
>
> `Priority` can be integer value in range 0-10, places with lower priority can be omitted to allow visiting more high-priority places.
>
> `StayDuration` is time that tourist plans to spend in place, will be used to calculate route optimizing for trip time and priorities. 

## Making requests

To make request and get complete response with headers one can use:

```bash
curl -v -H "Content-Type: application/json" -d @request.json http://localhost:8080/api/trip/
```

And for just pretty printed JSON:

```bash
curl -s -H "Content-Type: application/json" -d @request.json http://localhost:8080/api/trip/ | json_pp
```

# RESPONSE
