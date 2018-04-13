package simpletypes

import "io"

type String string

type StringAlias = String

type Map map[int]float64

type Array []bool

type Chan chan<- int

type Func func() error

type IntPointer *int

type RedundantPar (bool)

type Reader io.Reader
