// Package http provides a HTTP client for the string service.
package http

import (
	"net/url"
	"strings"
	
	httptransport "github.com/go-kit/kit/transport/http"
	
	"../gotravelsvc"
)

// New returns Service based on HTTP server at remote instance.
// Instance is expected to come in "host:port" form.
func New(instance string) gotravelsvc.Service {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		panic(err)
	}
	var tripPlanEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/trip"),
		gotravelsvc.EncodeRequest,
		gotravelsvc.DecodeTripPlanResponse,
	).Endpoint()
	
	return gotravelsvc.Endpoints{
		TripPlanEndpoint: tripPlanEndpoint,
	}
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}
