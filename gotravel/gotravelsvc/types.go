package gotravelsvc

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"sync"
	"time"
)

type Used map[int]bool
type Places map[int]*TripPlace

func NewPlaces(tps []*TripPlace) Places {
	places := make(Places, len(tps))
	for _, tp := range tps {
		places[tp.Index] = tp
	}
	return places
}

type DistanceMatrix = mat.Dense

func NewDistanceMatrix(n int) *DistanceMatrix {
	return mat.NewDense(n, n, nil)
}

type PheromonesMatrix struct {
	matrix *mat.Dense
}

func NewPheromonesMatrix(n int, initial float64) *PheromonesMatrix {
	data := make([]float64, n*n)
	for i := range data {
		data[i] = initial
	}
	return &PheromonesMatrix{mat.NewDense(n, n, data)}
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

func (p *PheromonesMatrix) IntensifyAlong(path *Path, boost int, mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()
	for i := 0; i < path.Size()-1; i++ {
		p.AddAt(path.At(i), path.At(i+1), float64(boost))
	}
	if path.loop {
		p.AddAt(path.At(path.len-1), path.At(0), float64(boost))
	}
}

func (p *PheromonesMatrix) Evaporate(boost, iterations int, mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()
	d := float64(boost) / float64(iterations)
	rows, cols := p.matrix.Caps()
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			p.Set(r, c, math.Max(0.0, p.At(r, c)-d))
		}
	}
}

type TravelTimeMatrix struct {
	matrix *mat.Dense
}

func NewTravelTimeMatrix(n int) *TravelTimeMatrix {
	return &TravelTimeMatrix{mat.NewDense(n, n, nil)}
}

func (m *TravelTimeMatrix) Set(i, j int, duration time.Duration) {
	m.matrix.Set(i, j, float64(duration.Nanoseconds()))
}

func (m *TravelTimeMatrix) At(i, j int) time.Duration {
	return time.Duration(m.matrix.At(i, j))
}

func (m *TravelTimeMatrix) AtAs(i, j int, as time.Duration) float64 {
	return float64(m.matrix.At(i, j) / float64(as))
}

type Path struct {
	path  []int
	len   int
	loop  bool
	dummy bool
}

func NewPath(size int, loop bool) Path {
	return Path{make([]int, size), size, loop, false}
}

func NewDummyPath() Path {
	return Path{dummy: true}
}

func (p *Path) Set(i, value int) {
	if i >= p.len {
		panic("Array index out of bounds")
	}
	p.path[i] = value
}

func (p *Path) PathIndexes() []int {
	if p.loop {
		return append(p.path, p.path[0])
	} else {
		return p.path
	}
}

func (p *Path) At(i int) int {
	if i >= p.len {
		panic("Array index out of bounds")
	}
	return p.path[i]
}

func (p *Path) Size() int {
	return p.len
}

func (p *Path) TotalDistance(distances *DistanceMatrix) float64 {
	if p.dummy {
		return math.MaxFloat64
	}

	tot := float64(0.0)

	for i := 0; i < p.Size()-1; i++ {
		tot += distances.At(p.At(i), p.At(i+1))
	}
	if p.loop {
		tot += distances.At(p.At(p.len-1), p.At(0))
	}

	return tot
}
