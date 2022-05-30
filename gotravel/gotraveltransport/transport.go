package gotraveltransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"

	"github.com/radekwlsk/go-travel/gotravel/gotravelendpoint"
	"github.com/radekwlsk/go-travel/gotravel/gotravelservice"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

func MakeHTTPHandler(endpoints gotravelendpoint.Endpoints, logger log.Logger) http.Handler {
	m := http.NewServeMux()
	options := []httptransport.ServerOption{
		httptransport.ServerErrorLogger(logger),
		httptransport.ServerErrorEncoder(errorEncoder),
	}

	m.Handle("/api/trip/", httptransport.NewServer(
		endpoints.TripPlanEndpoint,
		decodeTripPlanRequest,
		encodeResponse,
		options...,
	))

	return m
}

func MakeHTTPClient(instance string) (gotravelservice.Service, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	var options []httptransport.ClientOption

	var tripPlanEndpoint endpoint.Endpoint
	{
		tripPlanEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, "/api/trip/"),
			encodeTripPlanRequest,
			decodeTripPlanResponse,
			options...,
		).Endpoint()
	}

	return gotravelendpoint.Endpoints{
		TripPlanEndpoint: tripPlanEndpoint,
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

func decodeTripPlanRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request gotravelendpoint.TripPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&request.TripConfiguration); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeTripPlanResponse(_ context.Context, resp *http.Response) (interface{}, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, errorDecoder(resp)
	}
	var response gotravelendpoint.TripPlanResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

func encodeTripPlanRequest(ctx context.Context, req *http.Request, request interface{}) error {
	// r.Methods("POST").Path("/api/trip/")
	req.Method, req.URL.Path = "POST", "/api/trip/"
	return encodeRequest(ctx, req, request)
}

type erroneousResponse interface {
	Error() error
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(erroneousResponse); ok && e.Error() != nil {
		errorEncoder(ctx, e.Error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func encodeRequest(_ context.Context, req *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(&buf)
	return nil
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		panic("encodeError with nil Error")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(errToStatus(err))
	json.NewEncoder(w).Encode(map[string]interface{}{
		"err": err.Error(),
	})
}

func errToStatus(err error) int {
	switch err {
	case
		gotravelservice.ErrAPIKeyEmpty,
		gotravelservice.ErrModeEmpty,
		gotravelservice.ErrTripStartEmpty,
		gotravelservice.ErrTripEndEmpty,
		gotravelservice.ErrNotEnoughPlaces,
		gotravelservice.ErrBadTimeFormat,
		gotravelservice.ErrBadTime,
		gotravelservice.ErrEndBeforeStart,
		gotravelservice.ErrTwoStartPlaces,
		gotravelservice.ErrTwoEndPlaces,
		gotravelservice.ErrBadMode,
		gotravelservice.ErrBadTravelMode:
		return http.StatusBadRequest
	}
	switch err.(type) {
	case
		gotravelservice.ErrBadDescription,
		gotravelservice.ErrDescriptionInaccurate:
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

type errorWrapper struct {
	Error string `json:"err"`
}
