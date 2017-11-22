package ants

import (
	"github.com/afrometal/go-travel/gotravel/gotravelsvc"
	"gonum.org/v1/gonum/mat"
	"time"
)

type Used map[int]bool
type Places map[int]*gotravelsvc.TripPlace
type Distance float32

type DistanceMatrix = mat.Dense

type PheromonesMatrix = mat.Dense

type TravelTimeMatrix struct {
	matrix *mat.Dense
}

func NewTravelTimeMatrix(r, c int) *TravelTimeMatrix {
	return &TravelTimeMatrix{mat.NewDense(r, c, nil)}
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
	path []int
	len  int
	loop bool
}

func NewPath(size int, loop bool) Path {
	return Path{make([]int, size), size, loop}
}

func (p *Path) Set(i, value int) {
	if i >= p.len {
		panic("Array index out of bounds")
	}
	p.path[i] = value
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
	tot := float64(0.0)

	for i := 0; i < p.Size()-1; i++ {
		tot += distances.At(p.At(i), p.At(i+1))
	}
	if p.loop {
		tot += distances.At(p.At(p.len-1), p.At(0))
	}

	return tot
}

func (p *Path) TotalTime(times *TravelTimeMatrix) time.Duration {
	tot := time.Duration(0)

	for i := 0; i < p.Size()-1; i++ {
		tot += times.At(p.At(i), p.At(i+1))
	}
	if p.loop {
		tot += times.At(p.At(p.len-1), p.At(0))
	}

	return tot
}
