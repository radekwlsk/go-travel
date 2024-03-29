package ants

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/radekwlsk/go-travel/gotravel/gotravelservice/trip"
	"gonum.org/v1/gonum/mat"
)

type PheromonesMatrix struct {
	matrix *mat.Dense
	mutex  sync.Mutex
}

func NewPheromonesMatrix(n int, initial float64, mutex sync.Mutex) *PheromonesMatrix {
	data := make([]float64, n*n)
	for i := range data {
		data[i] = initial
	}
	return &PheromonesMatrix{mat.NewDense(n, n, data), mutex}
}

func (p *PheromonesMatrix) Set(i, j int, v float64) {
	p.matrix.Set(i, j, v)
}

func (p *PheromonesMatrix) At(i, j int) float64 {
	return p.matrix.At(i, j)
}

func (p *PheromonesMatrix) AddAt(i, j int, value float64) {
	p.Set(i, j, p.At(i, j)+value)
}

func (p *PheromonesMatrix) IntensifyAlong(path trip.Path, pheromone float64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, step := range path.Steps {
		p.AddAt(step.From, step.To, pheromone)
	}
}

func (p *PheromonesMatrix) Evaporate(boost float64, iterations int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	d := boost / float64(iterations)
	rows, cols := p.matrix.Caps()
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			p.Set(r, c, math.Max(0.0, p.At(r, c)-d))
		}
	}
}

type timesMappedMatrices map[time.Time]*mat.Dense

type timesMappedMatrix struct {
	times    []time.Time
	matrices timesMappedMatrices
}

func newTimesMappedMatrix(n int, times []time.Time) timesMappedMatrix {
	matrices := make(timesMappedMatrices, len(times))
	for _, t := range times {
		matrices[t] = mat.NewDense(n, n, nil)
	}
	return timesMappedMatrix{times, matrices}
}

func (m *timesMappedMatrix) matrixClosestTo(t time.Time) *mat.Dense {
	closest := m.times[0]
	if len(m.times) > 1 {
		diff := absTimeDifference(t, closest)
		for i, t2 := range m.times[1:] {
			d := absTimeDifference(t, t2)
			if d < diff {
				closest = m.times[i]
			}
		}
	}
	return m.matrices[closest]
}

type TimesMappedDurationsMatrix struct {
	timesMappedMatrix
}

func NewTravelTimeMatrix(n int, times []time.Time) *TimesMappedDurationsMatrix {
	return &TimesMappedDurationsMatrix{
		newTimesMappedMatrix(n, times),
	}
}

func absTimeDifference(t1 time.Time, t2 time.Time) time.Duration {
	if t1.After(t2) {
		return t1.Sub(t2)
	} else {
		return t2.Sub(t1)
	}
}

func (m *TimesMappedDurationsMatrix) Set(i, j int, t time.Time, duration time.Duration) {
	m.matrices[t].Set(i, j, float64(duration.Nanoseconds()))
}

func (m *TimesMappedDurationsMatrix) At(i, j int, t time.Time) time.Duration {
	if i == j {
		panic(errors.New("can not travel between the same place"))
	}
	return time.Duration(m.matrixClosestTo(t).At(i, j))
}

func (m *TimesMappedDurationsMatrix) AtAs(i, j int, t time.Time, as time.Duration) float64 {
	if i == j {
		panic(errors.New("can not travel between the same place"))
	}
	return float64(m.matrixClosestTo(t).At(i, j) / float64(as))
}

type TimesMappedDistancesMatrix struct {
	timesMappedMatrix
}

func NewDistanceMatrix(n int, times []time.Time) *TimesMappedDistancesMatrix {
	return &TimesMappedDistancesMatrix{
		newTimesMappedMatrix(n, times),
	}
}

func (m *TimesMappedDistancesMatrix) Set(i, j int, t time.Time, value int64) {
	m.matrices[t].Set(i, j, float64(value))
}

func (m *TimesMappedDistancesMatrix) At(i, j int, t time.Time) int64 {
	if i == j {
		panic(errors.New("can not travel between the same place"))
	}
	return int64(m.matrixClosestTo(t).At(i, j))
}
