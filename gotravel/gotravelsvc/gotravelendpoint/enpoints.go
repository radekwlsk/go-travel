package gotravelendpoint

import (
	"context"

	"github.com/afrometal/go-travel/gotravel/gotravelsvc/gotravelservice"
	"github.com/afrometal/go-travel/gotravel/gotravelsvc/gotravelservice/trip"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
)

type Endpoints struct {
	TripPlanEndpoint endpoint.Endpoint
}

func New(s gotravelservice.Service, logger log.Logger) Endpoints {
	var tripPlanEndpoint endpoint.Endpoint
	{
		tripPlanEndpoint = NewTripPlanEndpoint(s)
		tripPlanEndpoint = NewLoggingMiddleware(log.With(logger, "layer", "endpoint"))(tripPlanEndpoint)
	}
	return Endpoints{
		TripPlanEndpoint: tripPlanEndpoint,
	}
}

func (e Endpoints) TripPlan(ctx context.Context, tc trip.Configuration) (trip.Trip, error) {
	response, err := e.TripPlanEndpoint(ctx, TripPlanRequest{TripConfiguration: tc})
	if err != nil {
		return trip.Trip{}, err
	}
	resp := response.(TripPlanResponse)
	return resp.Trip, resp.Err
}

func NewTripPlanEndpoint(s gotravelservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(TripPlanRequest)
		resp, e := s.TripPlan(ctx, req.TripConfiguration)
		return TripPlanResponse{Trip: resp, Err: e}, nil
	}
}

type TripPlanRequest struct {
	TripConfiguration trip.Configuration
}

type TripPlanResponse struct {
	trip.Trip
	Err error `json:"err,omitempty"`
}

func (r TripPlanResponse) Error() error { return r.Err }
