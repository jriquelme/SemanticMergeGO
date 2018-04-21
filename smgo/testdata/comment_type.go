package commenttype

// some types
type ( // open (
	// string type
	String string // asdf

	// an alias
	StringAlias = String //
	// Map
	Map map[int]float64
	// Array
	Array []bool

	// person struct
	Person struct {
		Name string

		// Age
		Age int // field
	} // close }
	// Figures have area
	// and perimeter
	Figure interface {
		// Area
		Area() float64 // op1
		// Perimeter
		Perimeter() float64 // op2

		// before close }
	} // close }

	// before...
	// close )
) // close )

// empty type group
type () // nothing here

// channel type
type Chan chan<- int // chan!

// another struct
type AnotherStruct struct {
	// Func
	Func func() error // field1

	// pointer
	IntPointer *int // field2
}
