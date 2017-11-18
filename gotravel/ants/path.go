package ants

import "gonum.org/v1/gonum/mat"

type Path struct {
	path []int
	len  int
}

func NewPath(size int) *Path {
	return &Path{make([]int, size), size}
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

func (p *Path) Length(cities *mat.Dense) float64 {
	tot := float64(0.0)

	for i := 0; i < p.Size()-1; i++ {
		tot += cities.At(p.At(i), p.At(i+1))
	}

	tot += cities.At(p.At(p.Size()-1), p.At(0))
	return tot
}
