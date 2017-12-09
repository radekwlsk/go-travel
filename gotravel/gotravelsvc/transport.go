package gotravelsvc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"bytes"
	"errors"
	"io/ioutil"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

func MakeHTTPHandler(s Service, logger log.Logger) http.Handler {
	r := mux.NewRouter()
	e := MakeServerEndpoints(s)
	options := []httptransport.ServerOption{
		httptransport.ServerErrorLogger(logger),
		httptransport.ServerErrorEncoder(errorEncoder),
	}

	r.Methods("POST").Path("/api/trip/").Handler(httptransport.NewServer(
		e.TripPlanEndpoint,
		DecodeTripPlanRequest,
		EncodeResponse,
		options...,
	))

	return r
}

func DecodeTripPlanRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request tripPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&request.TripConfiguration); err != nil {
		return nil, err
	}
	return request, nil
}

func EncodeTripPlanRequest(ctx context.Context, req *http.Request, request interface{}) error {
	// r.Methods("POST").Path("/trip/")
	req.Method, req.URL.Path = "POST", "api/trip/"
	return EncodeRequest(ctx, req, request)
}

func DecodeTripPlanResponse(_ context.Context, resp *http.Response) (interface{}, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, errorDecoder(resp)
	}
	var response tripPlanResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}

type erroneousResponse interface {
	error() error
}

func EncodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(erroneousResponse); ok && e.error() != nil {
		errorEncoder(ctx, e.error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func EncodeRequest(_ context.Context, req *http.Request, request interface{}) error {
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
		panic("encodeError with nil error")
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
		ErrAPIKeyEmpty,
		ErrModeEmpty,
		ErrTripStartEmpty,
		ErrTripEndEmpty,
		ErrNotEnoughPlaces,
		ErrBadTimeFormat,
		ErrBadTime,
		ErrEndBeforeStart,
		ErrBadDescription,
		ErrTwoStartPlaces,
		ErrTwoEndPlaces,
		ErrBadMode,
		ErrBadTravelMode:
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

type errorWrapper struct {
	Error string `json:"err"`
}
