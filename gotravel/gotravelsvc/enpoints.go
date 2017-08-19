package gotravelsvc

import (
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"net/url"
	"strings"
	"context"
)

type Endpoints struct {
	TripPlanEndpoint   endpoint.Endpoint
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
		TripPlanEndpoint:   httptransport.NewClient("POST", tgt, EncodeTripPlanRequest, DecodeTripPlanResponse,
			options...).Endpoint(),
	}, nil
}

func (e Endpoints) TripPlan(ctx context.Context, tc TripConfiguration) (string, error) {
	request := tripPlanRequest{TripConfiguration: tc}
	response, err := e.TripPlanEndpoint(ctx, request)
	if err != nil {
		return "", err
	}
	resp := response.(tripPlanResponse)
	return resp.DecodedString, resp.Err
}

func MakeTripPlanEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(tripPlanRequest)
		str, e := s.TripPlan(ctx, req.TripConfiguration)
		return tripPlanResponse{DecodedString: str, Err: e}, nil
	}
}

type tripPlanRequest struct {
	TripConfiguration TripConfiguration
}

type tripPlanResponse struct {
	DecodedString string `json:"str,omitempty"`
	Err           error  `json:"err,omitempty"`
}

func (r tripPlanResponse) error() error { return r.Err }
