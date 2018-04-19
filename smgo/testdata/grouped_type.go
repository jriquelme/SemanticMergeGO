package groupedtype

import "io"

type (
	String string

	StringAlias = String
	Map         map[int]float64
	Array       []bool

	Chan chan<- int

	Func func() error

	IntPointer *int

	RedundantPar (bool)

	Reader io.Reader

	Person struct {
		Name string

		Age int
	}

	Figure interface {
		Area() float64
		Perimeter() float64
	}
)

type ()
