package gotravelsvc

import "googlemaps.github.io/maps"

type Planner struct {
	client *maps.Client
	trip   *Trip
}

func NewPlanner(c *maps.Client, t *Trip) *Planner {
	return &Planner{client: c, trip: t}
}

func (p *Planner) Evaluate() error {
	// TODO: implement algorithm here
	return nil
}
