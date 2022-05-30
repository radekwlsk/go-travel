# INSTALL

## Installing dependencies

1. Download and install Go following [go.dev/doc/install](https://go.dev/doc/install)
3. Create a directory structure `src/github.com/radekwlsk` in your `$GOPATH`
4. Clone this repository to `$(echo $GOPATH)/src/github.com/radekwlsk`
5. Go to `$(echo $GOPATH)/src/github.com/radekwlsk/go-travel` where `go.mod` file is located
6. Run `go mod vendor` to install local vendored copy of dependencies

## Install and run

1. Build and install the service with `go build gotravel/main.go`
2. Run with `./main`

or run server without installing it using `go run gotravel/main.go`

Server will be listening on port 8080 by default, change it by providing `-http-addr` argument.

# REQUESTS

## Format

Requests can be made in three modes:
1. **Address**, where places are identified by their address match on Google Maps.
3. **Name**, where places are searched on Google Maps by provided name and 1st result is selected.
4. **PlaceID**, where places are identified by their Google Maps API PlaceID

```
{
  "apiKey" : string,
  "mode": ["address"|"name"|"id"],
  "tripStart": string ("YYYY-MM-DDThh:mm:ssZ"),
  "tripEnd": string ("YYYY-MM-DDThh:mm:ssZ"),
  "language": string (2 letter code),
  "travelMode": ["driving", "walking", "transit", "bicycling"],
  "places": [
    {
      "description": {},
      "priority": int (0-10),
      "stayDuration": int (minutes)
    }
  ]
}
```
> `Description` mode specific place description used to identify specific location:
> - in **Address** mode:
>   ```json
>   {
>     "name": "Muzeum Narodowe we Wrocławiu",
>     "street": "plac Powstańców Warszawy",
>     "number": "5",
>     "city": "Wrocław",
>     "postalCode": "48-300",
>     "country": "Poland"
>   }
>   ```
>
> - in **Name** mode:
>   ```json
>   {
>     "name": "Muzeum Narodowe we Wrocławiu"
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

Examples for `address` and `name` modes are included as `address.json` and `name.json`. 

## Create API Key

Requesting data from Google Maps APIs requires providing API key that is activated for _Google Maps Places API_ and 
_Google Maps Distance Matrix API_. There may be free credit available but use of the APIs can generate cost. **Paying 
for and providing an API key lies is users responsibility.**

API Key has to be supplied as `"apiKey"` field in the request JSON.

Instructions how to acquire an API Key are available in 
[Google Maps Platform documentation](https://developers.google.com/maps/documentation/places/web-service/get-api-key).

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

```
{
  "schedule" : string,
  "totalDistance" : int (meters),
  "path" : [int],
  "steps" : [
     {
        "distance" : int (meters),
        "time" : int (minutes),
        "from" : int,
        "to" : int
     },
     ...
  ],
  "tripStart" : string ("YYYY-MM-DDThh:mm:ssZ"),
  "tripEnd" : string ("YYYY-MM-DDThh:mm:ssZ"),
  "travelMode" : ["driving", "walking", "transit", "bicycling"]
  "places" : [
     {
        "priority" : int (0-10),
        "details" : {
           "openingHours" : {
              "0" { "open" : string ("hhmm"), "close" : string ("hhmm") },
              "1" { "open" : string ("hhmm"), "close" : string ("hhmm") },
              "2" { "open" : string ("hhmm"), "close" : string ("hhmm") },
              "3" { "open" : string ("hhmm"), "close" : string ("hhmm") },
              "4" { "open" : string ("hhmm"), "close" : string ("hhmm") },
              "5" { "open" : string ("hhmm"), "close" : string ("hhmm") },
              "6" { "open" : string ("hhmm"), "close" : string ("hhmm") },
           },
           "name" : string,
           "formattedAddress" : string,
           "closed" : bool
        },
        "arrival" : string ("YYYY-MM-DDThh:mm:ssZ"),
        "departure" : string ("YYYY-MM-DDThh:mm:ssZ"),
        "id" : int,
        "stayDuration" : int (minutes)
     },
     ...
  ]
}
```
