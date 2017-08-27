package gotravelsvc

import (
	"context"
	"net/url"
	"strings"
	
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

type Endpoints struct {
	TripPlanEndpoint endpoint.Endpoint
}

func MakeServerEndpoints(s Service) Endpoints {
	return Endpoints{
		TripPlanEndpoint: MakeTripPlanEndpoint(s),
	}
}

func MakeClientEndpoints(instance string) (Endpoints, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	tgt, err := url.Parse(instance)
	if err != nil {
		return Endpoints{}, err
	}
	tgt.Path = ""
	
	options := []httptransport.ClientOption{}
	
	// Note that the request encoders need to modify the request URL, changing
	// the path and method. That's fine: we simply need to provide specific
	// encoders for each endpoint.
	
	return Endpoints{
		TripPlanEndpoint: httptransport.NewClient("POST", tgt, EncodeTripPlanRequest, DecodeTripPlanResponse,
			options...).Endpoint(),
	}, nil
}

func (e Endpoints) TripPlan(ctx context.Context, tc TripConfiguration) (Trip, error) {
	request := tripPlanRequest{TripConfiguration: tc}
	response, err := e.TripPlanEndpoint(ctx, request)
	if err != nil {
		return Trip{}, err
	}
	resp := response.(tripPlanResponse)
	return resp.Response, resp.Err
}

func MakeTripPlanEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(tripPlanRequest)
		resp, e := s.TripPlan(ctx, req.TripConfiguration)
		return tripPlanResponse{Response: resp, Err: e}, nil
	}
}

type tripPlanRequest struct {
	TripConfiguration TripConfiguration
}

type tripPlanResponse struct {
	Response Trip  `json:"resp,omitempty"`
	Err      error `json:"err,omitempty"`
}

func (r tripPlanResponse) error() error { return r.Err }
